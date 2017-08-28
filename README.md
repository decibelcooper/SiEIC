# Usage

## Running a basic workflow
### Getting the software image
It is recommended for one to use Singularity as a container environment.  Make
sure you have Singularity 2.3 or above installed to get started.  Also, make
sure you are in a directory with at least 6GB of space available.

```
singularity pull docker://argonneeic/fpadsim
singularity exec fpadsim.img bash -l
```

### Executing a workflow
After running the above commands, you should be in an environment similar to
the one you started with, but with a completely different set of system
software available.  One may then clone this repository and begin using it.

```shell
git clone https://github.com/decibelCooper/SiEIC.git
cd SiEIC
make
```

On the make command, one should see that needed files that are derivative of
files in the repository are built, such as geometry conversions.  If promc
generator output files are placed into the input directory or subdirectories
therein, make will also create simulation, reconstruction, and analysis targets
for each file.  One may draw from HepSim and begin the workflow by first
editing nEventsPerRun to contain a reasonable value (let's say 10), and running
the following commands.

```shell
cd input
wget http://mc.hep.anl.gov/asc/hepsim/events/misc/pgun/pgun_eta4_pt30_elec/pgun_elec30gev_001.promc
cd ..
make
```

On the make command, one should see the commands such as ddsim being run on the
downloaded promc file.

## Running a workflow on Bebop
The main tool specific to Bebop is `tools/bebop.submit`.  This is a shell script with extra configuration at the top for running the slurm `sbatch` command.  As in the above example, truth-level events in ProMC format must first be placed into the `input/` directory (or a subdirectory therein).  Then, the `tools/bebop.submit` file must be configured.  The beginning lines starting with `#SBATCH` are passed on to the sbatch command as arguments, and the sbatch man page can be referenced to for help with the arguments.  It is critical here to choose a number of nodes that in total has a number of CPU cores that meets or exceeds the number of ProMC files in the input directory.  It is also critical that the requested time is chosen to exceed the amount of time that a single core takes to run through the entire chain for the chosen number of `nEventsPerRun`.

Once the input files are acquired, and `tools/bebop.submit` and `nEventsPerRun` have been configured, one may run a batch job on Bebop with the command...
```shell
sbatch tools/bebop.submit
```

Please note that the `tools/bebop.submit` script assumes a particular path for the singularity image.  If it does not exist, it will create an image from the docker hub.  If you have already created an image for using `hs-get` for example, consider placing the image in the location that the script will look for one, in order to avoid creating duplicate images.
