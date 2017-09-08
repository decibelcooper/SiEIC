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
	inputsAreDirs = flag.Bool("d", false, "inputs are directories")
	maxFiles      = flag.Int("m", math.MaxInt32, "maximum number of files to process")
	nThreads      = flag.Int("t", 2, "number of concurrent files to process")
	outputPath    = flag.String("o", "out.pdf", "path of output file")
)

const (
	minEta            = -5
	maxEta            = 5
	nEtaBins          = 50
	truthChargedMinPT = 0.5
)

type ParticleType uint8

const (
	ELEC ParticleType = iota
	PION
	PROTON
	PHOTON
	NEUTRON
	OTHER
)

type Result struct {
	Charge float32
	Eta    float64
	Type   ParticleType
	Weight float64
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: PFODist [options] <lcio-input-file>
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
	p.Title.Text = "PFO/Truth Comparison"
	p.Title.Padding = 2 * vg.Millimeter
	p.X.Label.Text = "eta"
	p.Y.Label.Text = "count"
	p.Legend.Top = true
	p.Legend.Left = true
	p.Legend.Padding = 2 * vg.Millimeter
	if *inputsAreDirs {
		p.Title.Text = "PFO Comparison"
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

			var redTint uint8
			var dashes []vg.Length
			var dashOffs vg.Length
			switch i {
			case 0:
			case 1:
				redTint = 255
				dashes = append(dashes, 1*vg.Millimeter)
			}

			drawFileSet(inputFiles, p, false, redTint, path.Base(dir), dashes, dashOffs)
		}
	} else {
		drawFileSet(flag.Args(), p, true, 0, "PandoraPFO", nil, 0)
	}

	p.Save(6*vg.Inch, 4*vg.Inch, *outputPath)
}

func drawFileSet(inputFiles []string, p *hplot.Plot, drawTruth bool, histRedTint uint8, histLabelPrefix string, histDashes []vg.Length, histDashOffs vg.Length) {
	elecPFOEtaHist := hbook.NewH1D(nEtaBins, minEta, maxEta)
	elecTrueEtaHist := hbook.NewH1D(nEtaBins, minEta, maxEta)
	chargedPFOEtaHist := hbook.NewH1D(nEtaBins, minEta, maxEta)
	chargedTrueEtaHist := hbook.NewH1D(nEtaBins, minEta, maxEta)
	neutralPFOEtaHist := hbook.NewH1D(nEtaBins, minEta, maxEta)
	neutralTrueEtaHist := hbook.NewH1D(nEtaBins, minEta, maxEta)

	trueOut := make(chan Result)
	pfoOut := make(chan Result)
	done := make(chan bool)

	nFilesToAnalyze := len(inputFiles)
	if *maxFiles < nFilesToAnalyze {
		nFilesToAnalyze = *maxFiles
	}

	nSubmitted := 0
	nDone := 0

	for nSubmitted < nFilesToAnalyze && nSubmitted < *nThreads {
		go analyzeFile(inputFiles[nSubmitted], trueOut, pfoOut, done)
		nSubmitted++

		time.Sleep(time.Millisecond)
	}

	for nDone < nSubmitted {
		select {
		case result := <-trueOut:
			if result.Charge != 0 {
				chargedTrueEtaHist.Fill(result.Eta, result.Weight)
			} else {
				neutralTrueEtaHist.Fill(result.Eta, result.Weight)
			}

			if result.Type == ELEC {
				elecTrueEtaHist.Fill(result.Eta, result.Weight)
			}
		case result := <-pfoOut:
			if result.Charge != 0 {
				chargedPFOEtaHist.Fill(result.Eta, result.Weight)
			} else {
				neutralPFOEtaHist.Fill(result.Eta, result.Weight)
			}

			if result.Type == ELEC {
				elecPFOEtaHist.Fill(result.Eta, result.Weight)
			}
		case isDone := <-done:
			if isDone {
				nDone++

				if nSubmitted < nFilesToAnalyze {
					go analyzeFile(inputFiles[nSubmitted], trueOut, pfoOut, done)
					nSubmitted++
				}
			}
		}
	}

	/*
		hElecTrue, err := hplot.NewH1D(elecTrueEtaHist)
		if err != nil {
			panic(err)
		}
		hElecTrue.LineStyle.Color = color.RGBA{R: 255, A: 255, G: 150, B: 150}
		hElecTrue.FillColor = nil
		p.Add(hElecTrue)

		hElecPFO, err := hplot.NewH1D(elecPFOEtaHist)
		if err != nil {
			panic(err)
		}
		hElecPFO.LineStyle.Color = color.RGBA{R: 255, A: 255}
		hElecPFO.FillColor = nil
		p.Add(hElecPFO)
	*/

	if drawTruth {
		hChargedTrue, err := hplot.NewH1D(chargedTrueEtaHist)
		if err != nil {
			panic(err)
		}
		hChargedTrue.LineStyle.Color = color.RGBA{B: 255, A: 255, R: 150, G: 150}
		hChargedTrue.FillColor = nil
		p.Add(hChargedTrue)
		p.Legend.Add("MCParticle Charged", hChargedTrue)
	}

	hChargedPFO, err := hplot.NewH1D(chargedPFOEtaHist)
	if err != nil {
		panic(err)
	}
	hChargedPFO.LineStyle.Color = color.RGBA{B: 255, A: 255, R: histRedTint}
	hChargedPFO.LineStyle.Dashes = histDashes
	hChargedPFO.LineStyle.DashOffs = histDashOffs
	hChargedPFO.FillColor = nil
	p.Add(hChargedPFO)
	p.Legend.Add(histLabelPrefix+" Charged", hChargedPFO)

	if drawTruth {
		hNeutralTrue, err := hplot.NewH1D(neutralTrueEtaHist)
		if err != nil {
			panic(err)
		}
		hNeutralTrue.LineStyle.Color = color.RGBA{G: 255, A: 255, R: 150, B: 150}
		hNeutralTrue.FillColor = nil
		p.Add(hNeutralTrue)
		p.Legend.Add("MCParticle Neutral", hNeutralTrue)
	}

	hNeutralPFO, err := hplot.NewH1D(neutralPFOEtaHist)
	if err != nil {
		panic(err)
	}
	hNeutralPFO.LineStyle.Color = color.RGBA{G: 255, A: 255, R: histRedTint}
	hNeutralPFO.LineStyle.Dashes = histDashes
	hNeutralPFO.LineStyle.DashOffs = histDashOffs
	hNeutralPFO.FillColor = nil
	p.Add(hNeutralPFO)
	p.Legend.Add(histLabelPrefix+" Neutral", hNeutralPFO)
}

func analyzeFile(inputPath string, trueOut chan<- Result, pfoOut chan<- Result, done chan<- bool) {
	reader, err := lcio.Open(inputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	for reader.Next() {
		event := reader.Event()

		truthColl := event.Get("MCParticle").(*lcio.McParticleContainer)
		pfoColl := event.Get("PandoraPFOCollection").(*lcio.RecParticleContainer)

		for _, truth := range truthColl.Particles {
			if truth.GenStatus != 1 {
				continue
			}

			pNorm := normalizeVector(truth.P)
			eta := math.Atanh(pNorm[2])

			absPDG := truth.PDG
			if absPDG < 0 {
				absPDG = -absPDG
			}

			particleType := OTHER
			switch absPDG {
			case 11:
				particleType = ELEC
			case 111:
				fallthrough
			case 211:
				particleType = PION
			case 2212:
				particleType = PROTON
			case 22:
				particleType = PHOTON
			case 2112:
				particleType = NEUTRON
			}

			trueOut <- Result{truth.Charge, eta, particleType, 1}
		}

		for _, pfo := range pfoColl.Parts {
			pNorm := normalizeVector32(pfo.P)
			eta := math.Atanh(pNorm[2])

			absPDG := pfo.Type
			if absPDG < 0 {
				absPDG = -absPDG
			}

			particleType := OTHER
			switch absPDG {
			case 11:
				particleType = ELEC
			case 111:
				fallthrough
			case 211:
				particleType = PION
			case 2212:
				particleType = PROTON
			case 22:
				particleType = PHOTON
			case 2112:
				particleType = NEUTRON
			}

			pfoOut <- Result{pfo.Charge, eta, particleType, 1}
		}
	}

	done <- true
}

func normalizeVector32(vector [3]float32) (result [3]float64) {
	normFactor := math.Sqrt(dotProduct32(vector, vector))
	for i, value := range vector {
		result[i] = float64(value) / normFactor
	}
	return
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

func dotProduct32(vector1 [3]float32, vector2 [3]float32) float64 {
	return float64(vector1[0]*vector2[0] + vector1[1]*vector2[1] + vector1[2]*vector2[2])
}

func dotProduct(vector1 [3]float64, vector2 [3]float64) float64 {
	return vector1[0]*vector2[0] + vector1[1]*vector2[1] + vector1[2]*vector2[2]
}
