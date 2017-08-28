# Usage

## Getting the software chain image

It is recommended for one to use Singularity as a container environment.  Make
sure you have Singularity 2.3 or above installed to get started.  Also, make
sure you are in a directory with at least 6GB of space available.

```
singularity create -s 6000 fpadsim.img
singularity import fpadsim.img docker://argonneeic/fpadsim
singularity exec fpadsim.img bash -l
```

## Running a workflow

After running the above commands, you should be in an environment similar to
the one you started with, but with a completely different set of system
software available.  One may then clone this repository and begin using it.

```
git clone https://github.com/decibelCooper/FPaDAnalysis.git
cd FPaDAnalysis
make
```

On the make command, one should see that needed files that are derivative of
files in the repository are built, such as geometry conversions.  If promc
generator output files are placed into the input directory or subdirectories
therein, make will also create simulation, reconstruction, and analysis targets
for each file.  One may draw from HepSim and begin the workflow by first
editing nEventsPerRun to contain a reasonable value (let's say 10), and running
the following commands.

```
cd input
wget http://mc.hep.anl.gov/asc/hepsim/events/misc/pgun/pgun_eta4_pt30_elec/pgun_elec30gev_001.promc
cd ..
make
```

On the make command, one should see the commands such as ddsim being run on the
downloaded promc file.
