#!/bin/bash
#SBATCH -p bdwall
#SBATCH -N 14
#SBATCH --time=1:00:00

singularityImage="../fpadsim-v1.4.img"
files=$(find input -iname "*.promc")

node=0
function runTaskOnNode {
	srun -lN1 -r$node singularity exec $singularityImage bash -lc "make -j$(($SLURM_CPUS_ON_NODE + 1)) $1" &
	node=$(($node + 1))
}

module load singularity

singularity exec $singularityImage bash -lc "make -j4 init"

i=0
for file in $files; do
	targets="$targets $(echo $file | sed 's/^input\//output\//' | sed 's/\.promc/_hepsim.slcio/')"

	i=$(($i + 1))

	if [ "$(($i % $SLURM_CPUS_ON_NODE))" == "0" ]; then

		runTaskOnNode "$targets"
		targets=""
	fi
done

if [ "$targets" != "" ]; then
	runTaskOnNode "$targets"
fi

wait
