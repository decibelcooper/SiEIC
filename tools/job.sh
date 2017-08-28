#!/bin/bash -l

mkdir -p input/$1 output
mv $(basename $2) input/$1/

make osg

