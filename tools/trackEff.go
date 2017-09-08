package main

import (
	"flag"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"time"

	"go-hep.org/x/hep/hbook"
	"go-hep.org/x/hep/hplot"
	"go-hep.org/x/hep/lcio"

	"gonum.org/v1/plot/vg"
)

var (
	doMinAnglePlot   = flag.Bool("a", false, "generate plot of minimum angle between Tracks and MCParticles")
	inputsAreDirs    = flag.Bool("d", false, "inputs are directories")
	maxFiles         = flag.Int("m", math.MaxInt32, "maximum number of files to process")
	normalize        = flag.Bool("n", false, "normalize Track count to MCParticle count")
	nThreads         = flag.Int("t", 2, "number of concurrent files to process")
	outputPath       = flag.String("o", "out.pdf", "path of output file")
	showTrackSummary = flag.Bool("s", false, "show stats summary for track distribution")
	vsP_T            = flag.Bool("p", false, "plot efficiency vs. p_T")
)

const (
	maxAngle   = 0.01
	minEta     = -5
	maxEta     = 5
	minP_T     = 0.5
	maxP_T     = 5
	nEtaBins   = 50
	nAngleBins = 50
	nP_TBins   = 50
	truthMinPT = 0.5
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: trackEff [options] <lcio-input-file>
options:
`,
		)
		flag.PrintDefaults()
	}

	flag.Parse()

	p, err := hplot.New()
	if err != nil {
		panic(err)
	}

	p.Title.Text = "Tracking/Truth Comparison"
	if *normalize {
		p.Title.Text = "Tracking Efficiency"
	} else if *inputsAreDirs {
		p.Title.Text = "Tracking Comparison"
	}
	p.Title.Padding = 2 * vg.Millimeter
	p.Legend.Left = true
	p.Legend.Top = true
	p.Legend.Padding = 2 * vg.Millimeter
	p.X.Label.Text = "eta"
	if *doMinAnglePlot {
		p.Legend.Left = false
		p.X.Label.Text = "min. angular deviation"
	} else if *vsP_T {
		p.X.Label.Text = "p_T {GeV}"
	}

	if *inputsAreDirs {
		for i, dir := range flag.Args() {
			files, err := ioutil.ReadDir(dir)
			if err != nil {
				log.Fatal(err)
			}

			var inputFiles []string
			for _, file := range files {
				inputFiles = append(inputFiles, dir+"/"+file.Name())
			}

			trackColor := color.RGBA{R: 255, B: 255, G: 255, A: 255}
			var dashes []vg.Length
			var dashOffs vg.Length
			switch i {
			case 0:
				trackColor = color.RGBA{B: 255, A: 255}
			case 1:
				trackColor = color.RGBA{R: 255, A: 255}
				dashes = append(dashes, 1*vg.Millimeter)
			case 2:
				trackColor = color.RGBA{G: 255, A: 255}
				dashes = append(dashes, 1*vg.Millimeter)
				dashOffs = 1 * vg.Millimeter
			}

			drawFileSet(inputFiles, p, false, trackColor, path.Base(dir), dashes, dashOffs)
		}
	} else {
		histColor := color.RGBA{R: 255, A: 255}
		if *normalize || *doMinAnglePlot {
			histColor = color.RGBA{B: 255, A: 255}
		}

		drawFileSet(flag.Args(), p, true, histColor, "Track", nil, 0)
	}

	p.Save(6*vg.Inch, 4*vg.Inch, *outputPath)
}

type TrueResult struct {
	Eta float64
	P_T float64
}

type TrackResult struct {
	MinAngle float64
	Eta      float64
	P_T      float64
}

func drawFileSet(inputFiles []string, p *hplot.Plot, drawTruth bool, trackColor color.Color, trackLabel string, trackDashes []vg.Length, trackDashOffs vg.Length) {
	trueEtaHist := hbook.NewH1D(nEtaBins, minEta, maxEta)
	trackEtaHist := hbook.NewH1D(nEtaBins, minEta, maxEta)
	minAngleHist := hbook.NewH1D(nAngleBins, 0, maxAngle)
	trueP_THist := hbook.NewH1D(nP_TBins, minP_T, maxP_T)
	trackP_THist := hbook.NewH1D(nP_TBins, minP_T, maxP_T)

	trueResults := make(chan TrueResult)
	trackResults := make(chan TrackResult)
	done := make(chan bool)

	nFilesToAnalyze := len(inputFiles)
	if *maxFiles < nFilesToAnalyze {
		nFilesToAnalyze = *maxFiles
	}

	nSubmitted := 0
	nDone := 0

	for nSubmitted < nFilesToAnalyze && nSubmitted < *nThreads {
		go analyzeFile(inputFiles[nSubmitted], trueResults, trackResults, done)
		nSubmitted++

		time.Sleep(time.Millisecond)
	}

	for nDone < nSubmitted {
		select {
		case trueResult := <-trueResults:
			trueEtaHist.Fill(trueResult.Eta, 1)
			trueP_THist.Fill(trueResult.P_T, 1)
		case trackResult := <-trackResults:
			trackEtaHist.Fill(trackResult.Eta, 1)
			minAngleHist.Fill(trackResult.MinAngle, 1)
			trackP_THist.Fill(trackResult.P_T, 1)
		case <-done:
			nDone++

			if nSubmitted < nFilesToAnalyze {
				go analyzeFile(inputFiles[nSubmitted], trueResults, trackResults, done)
				nSubmitted++
			}
		}
	}

	if *doMinAnglePlot {
		h, err := hplot.NewH1D(minAngleHist)
		if err != nil {
			panic(err)
		}
		h.LineStyle.Color = trackColor
		h.LineStyle.Dashes = trackDashes
		h.LineStyle.DashOffs = trackDashOffs
		p.Add(h)
		if *inputsAreDirs {
			p.Legend.Add(trackLabel, h)
		}
	} else {
		var hTrue *hplot.H1D
		var err error
		if drawTruth {
			if !*vsP_T {
				hTrue, err = hplot.NewH1D(trueEtaHist)
			} else {
				hTrue, err = hplot.NewH1D(trueP_THist)
			}
			if err != nil {
				panic(err)
			}
			hTrue.LineStyle.Color = color.RGBA{B: 255, A: 255}
			if !*normalize {
				p.Add(hTrue)
				p.Legend.Add("MCParticle", hTrue)
			}
		}

		var hTrack *hplot.H1D
		if !*vsP_T {
			hTrack, err = hplot.NewH1D(trackEtaHist)
		} else {
			hTrack, err = hplot.NewH1D(trackP_THist)
		}
		if err != nil {
			panic(err)
		}
		hTrack.LineStyle.Color = trackColor
		hTrack.LineStyle.Dashes = trackDashes
		hTrack.LineStyle.DashOffs = trackDashOffs
		if *showTrackSummary {
			hTrack.Infos.Style = hplot.HInfoSummary
		}
		if !*normalize {
			p.Add(hTrack)
			p.Legend.Add(trackLabel, hTrack)
		}

		normHist := hbook.NewH1D(hTrue.Hist.Len(), hTrue.Hist.XMin(), hTrue.Hist.XMax())
		if *normalize {
			for i := 0; i < normHist.Len(); i++ {
				trueX, trueY := hTrue.Hist.XY(i)
				_, trackY := hTrack.Hist.XY(i)
				if trueY > 0 {
					normHist.Fill(trueX, trackY/trueY)
				}
			}

			hNorm, err := hplot.NewH1D(normHist)
			if err != nil {
				panic(err)
			}
			hNorm.LineStyle.Color = trackColor
			hNorm.LineStyle.Dashes = trackDashes
			hNorm.LineStyle.DashOffs = trackDashOffs
			p.Add(hNorm)
			if *inputsAreDirs {
				p.Legend.Add(trackLabel, hNorm)
			}
		}
	}
}

type TruthRelation struct {
	Truth *lcio.McParticle
	PNorm [3]float64
	Eta   float64
	P_T   float64
}

func analyzeFile(inputPath string, trueResults chan<- TrueResult, trackResults chan<- TrackResult, done chan<- bool) {
	reader, err := lcio.Open(inputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	for reader.Next() {
		event := reader.Event()

		truthColl := event.Get("MCParticle").(*lcio.McParticleContainer)
		trackColl := event.Get("Tracks").(*lcio.TrackContainer)

		// FIXME: boost back from crossing angle?

		var truthRelations []TruthRelation
		for i, truth := range truthColl.Particles {
			if truth.GenStatus != 1 || truth.Charge == float32(0) {
				continue
			}

			pNorm := normalizeVector(truth.P)
			eta := math.Atanh(pNorm[2])
			pT := math.Sqrt(truth.P[0]*truth.P[0] + truth.P[1]*truth.P[1])

			if pT > truthMinPT {
				truthRelations = append(truthRelations, TruthRelation{
					Truth: &truthColl.Particles[i],
					PNorm: pNorm,
					Eta:   eta,
					P_T:   pT,
				})

				trueResults <- TrueResult{
					Eta: eta,
					P_T: pT,
				}
			}
		}

		for _, track := range trackColl.Tracks {
			tanLambda := track.TanL()
			//eta := -math.Log(math.Sqrt(1+tanLambda*tanLambda) - tanLambda)

			lambda := math.Atan(tanLambda)
			px := math.Cos(track.Phi()) * math.Cos(lambda)
			py := math.Sin(track.Phi()) * math.Cos(lambda)
			pz := math.Sin(lambda)

			pNorm := [3]float64{px, py, pz}

			minAngle := math.Inf(1)
			minIndex := -1
			for i, truthRelation := range truthRelations {
				angle := math.Acos(dotProduct(pNorm, truthRelation.PNorm))
				if angle < minAngle {
					minAngle = angle
					minIndex = i
				}
			}

			if minIndex >= 0 && minAngle < maxAngle {
				trackResults <- TrackResult{
					MinAngle: minAngle,
					Eta:      truthRelations[minIndex].Eta,
					P_T:      truthRelations[minIndex].P_T,
				}

				truthRelations = append(truthRelations[:minIndex], truthRelations[minIndex+1:]...)
			}
		}
	}

	done <- true
}

func normalizeVector(vector [3]float64) [3]float64 {
	normFactor := math.Sqrt(dotProduct(vector, vector))
	for i, value := range vector {
		vector[i] = value / normFactor
	}
	return vector
}

func phiFromVector(vector [3]float64) float64 {
	rho := math.Sqrt(vector[0]*vector[0] + vector[1]*vector[1])
	if vector[0] >= 0 {
		return math.Asin(vector[1] / rho)
	}
	return -math.Asin(vector[1]/rho) + math.Pi
}

func dotProduct(vector1 [3]float64, vector2 [3]float64) float64 {
	return vector1[0]*vector2[0] + vector1[1]*vector2[1] + vector1[2]*vector2[2]
}
