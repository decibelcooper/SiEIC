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
	energyWeighted = flag.Bool("e", false, "weight distribution by energy")
	inputsAreDirs  = flag.Bool("d", false, "inputs are directories")
	maxFiles       = flag.Int("m", math.MaxInt32, "maximum number of files to process")
	nThreads       = flag.Int("t", 2, "number of concurrent files to process")
	outputPath     = flag.String("o", "out.pdf", "path of output file")
)

const (
	minEta   = -5
	maxEta   = 5
	nEtaBins = 50
)

type clusterResult struct {
	Eta    float64
	Energy float64
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: clusterDist [options] <lcio-input-file>
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

	p.Title.Text = "Cluster Distribution"
	p.Title.Padding = 2 * vg.Millimeter
	p.Legend.Left = true
	p.Legend.Top = true
	p.Legend.Padding = 2 * vg.Millimeter
	p.X.Label.Text = "eta"
	if *energyWeighted {
		p.Y.Label.Text = "energy (arb)"
	} else {
		p.Y.Label.Text = "count"
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

			histColor := color.RGBA{R: 255, B: 255, G: 255, A: 255}
			var dashes []vg.Length
			var dashOffs vg.Length
			switch i {
			case 0:
				histColor = color.RGBA{B: 255, A: 255}
			case 1:
				histColor = color.RGBA{R: 255, A: 255}
				dashes = append(dashes, 1*vg.Millimeter)
			case 2:
				histColor = color.RGBA{G: 255, A: 255}
				dashes = append(dashes, 1*vg.Millimeter)
				dashOffs = 1 * vg.Millimeter
			}

			drawFileSet(inputFiles, p, histColor, path.Base(dir), dashes, dashOffs)
		}
	} else {
		drawFileSet(flag.Args(), p, color.RGBA{B: 255, A: 255}, "", nil, 0)
	}

	p.Save(6*vg.Inch, 4*vg.Inch, *outputPath)
}

func drawFileSet(inputFiles []string, p *hplot.Plot, histColor color.Color, histLabel string, histDashes []vg.Length, histDashOffs vg.Length) {
	clusterEtaHist := hbook.NewH1D(nEtaBins, minEta, maxEta)

	clusterOut := make(chan clusterResult)
	done := make(chan bool)

	nFilesToAnalyze := len(inputFiles)
	if *maxFiles < nFilesToAnalyze {
		nFilesToAnalyze = *maxFiles
	}

	nSubmitted := 0
	nDone := 0

	for nSubmitted < nFilesToAnalyze && nSubmitted < *nThreads {
		go analyzeFile(inputFiles[nSubmitted], clusterOut, done)
		nSubmitted++

		time.Sleep(time.Millisecond)
	}

	for nDone < nSubmitted {
		select {
		case result := <-clusterOut:
			clusterEtaHist.Fill(result.Eta, result.Energy)
		case isDone := <-done:
			if isDone {
				nDone++

				if nSubmitted < nFilesToAnalyze {
					go analyzeFile(inputFiles[nSubmitted], clusterOut, done)
					nSubmitted++
				}
			}
		}
	}

	hCluster, err := hplot.NewH1D(clusterEtaHist)
	if err != nil {
		panic(err)
	}
	hCluster.LineStyle.Color = histColor
	hCluster.LineStyle.Dashes = histDashes
	hCluster.LineStyle.DashOffs = histDashOffs
	p.Add(hCluster)
	if *inputsAreDirs {
		p.Legend.Add(histLabel, hCluster)
	}
}

func analyzeFile(inputPath string, clusterOut chan<- clusterResult, done chan<- bool) {
	reader, err := lcio.Open(inputPath)
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	for reader.Next() {
		event := reader.Event()

		clusterColl := event.Get("ReconClusters").(*lcio.ClusterContainer)

		for _, cluster := range clusterColl.Clusters {
			pNorm := normalizePos(cluster.Pos)
			eta := math.Atanh(pNorm[2])
			energy := 1.
			if *energyWeighted {
				energy = float64(cluster.Energy)
			}

			result := clusterResult{eta, energy}
			clusterOut <- result
		}
	}

	done <- true
}

func normalizePos(vector [3]float32) (retVec [3]float64) {
	normFactor := math.Sqrt(dotProduct32(vector, vector))
	for i, value := range vector {
		retVec[i] = float64(value) / normFactor
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
