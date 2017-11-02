# Require second expansion for tricksy $$() substitutions
.SECONDEXPANSION:

# Define geometry target paths
# FIXME: move GEOM_BASE definition outside of Makefile
GEOM_BASE = sieic6
GEOM_PATH = geom
GEOM_LCDD = $(GEOM_PATH)/$(GEOM_BASE).lcdd
GEOM_HEPREP = $(GEOM_PATH)/$(GEOM_BASE).heprep
GEOM_GDML = $(GEOM_PATH)/$(GEOM_BASE).gdml
GEOM_PANDORA = $(GEOM_PATH)/$(GEOM_BASE).pandora
GEOM_HTML = $(GEOM_PATH)/$(GEOM_BASE).html
LCSIM_CONDITIONS_PREFIX := http%3A%2F%2Fwww.lcsim.org%2Fdetectors%2F
LCSIM_CONDITIONS_PREFIX_ESCAPED := http\%3A\%2F\%2Fwww.lcsim.org\%2Fdetectors\%2F
LCSIM_CONDITIONS := $(PWD)/.lcsim/cache/$(LCSIM_CONDITIONS_PREFIX)$(GEOM_BASE).zip
GEOM_OVERLAP_CHECK = $(GEOM_PATH)/overlapCheck.log
GEOM = $(GEOM_LCDD) $(GEOM_GDML) $(GEOM_HEPREP) $(GEOM_PANDORA) $(GEOM_HTML) $(LCSIM_CONDITIONS) \
	$(GEOM_OVERLAP_CHECK)

# Define tracking strategy list target path
STRATEGIES = $(GEOM_PATH)/config/trackingStrategies.xml

# Grab number of events to simulate
N_EVENTS = $(shell cat nEventsPerRun)

# Create output target file paths for each input file
INPUT_BASE = $(patsubst input/%,%,$(basename $(shell find input -iname "*.promc")))
OUTPUT_DIRS = $(sort $(dir $(patsubst input/%,output/%,$(basename $(shell find input -iname "*.promc")))) $(sort $(dir $(wildcard output/*/))))

OUTPUT_TRUTH = $(addprefix output/,$(INPUT_BASE:=_truth.slcio))
OUTPUT_SIM = $(addprefix output/,$(INPUT_BASE:=.slcio))
OUTPUT_TRACKING = $(addprefix output/,$(INPUT_BASE:=_tracking.slcio))
OUTPUT_TRACKING_PROIO = $(addprefix output/,$(INPUT_BASE:=_tracking.proio.gz))
OUTPUT_PANDORA = $(addprefix output/,$(INPUT_BASE:=_pandora.slcio))
OUTPUT_HEPSIM = $(addprefix output/,$(INPUT_BASE:=_hepsim.slcio))

OUTPUT_TRACKEFF = $(OUTPUT_DIRS:=trackEff.pdf)
OUTPUT_TRACKEFF_NORM = $(OUTPUT_DIRS:=trackEff-norm.pdf)
OUTPUT_TRACKEFF_DEVANG = $(OUTPUT_DIRS:=trackEff-devAng.pdf)
OUTPUT_TRACKEFF_PT = $(OUTPUT_DIRS:=trackEff-pT.pdf)
OUTPUT_TRACKEFF_PT_NORM = $(OUTPUT_DIRS:=trackEff-pT-norm.pdf)
OUTPUT_CLUSTERDIST = $(OUTPUT_DIRS:=clusterDist.pdf)
OUTPUT_CLUSTERDIST_EWEIGHT = $(OUTPUT_DIRS:=clusterDist-energyWeighted.pdf)
OUTPUT_PFODIST = $(OUTPUT_DIRS:=pfoDist.pdf)
OUTPUT_DIAG = $(OUTPUT_TRACKEFF_DEVANG) $(OUTPUT_TRACKEFF) $(OUTPUT_TRACKEFF_NORM) $(OUTPUT_TRACKEFF_PT) $(OUTPUT_TRACKEFF_PT_NORM) \
			  $(OUTPUT_CLUSTERDIST) $(OUTPUT_CLUSTERDIST_EWEIGHT) \
			  $(OUTPUT_PFODIST)

# Set what output files to build by default
OUTPUT = $(OUTPUT_TRUTH) $(OUTPUT_SIM) $(OUTPUT_TRACKING) $(OUTPUT_TRACKING_PROIO)
ifeq ($(MAKECMDGOALS),hepsim)
.INTERMEDIATE: $(OUTPUT_TRUTH) $(OUTPUT_SIM) $(OUTPUT_TRACKING) $(OUTPUT_PANDORA)
endif

.PHONY: all init hepsim sim clean allclean

all: $(OUTPUT) $(GEOM) $(STRATEGIES)

init: $(GEOM) $(STRATEGIES)

hepsim: $(OUTPUT_HEPSIM)

sim: $(OUTPUT_SIM)

clean:
	rm -rf output/*

initclean:
	rm -rf $(GEOM) $(STRATEGIES)

allclean:
	rm -rf output/* $(GEOM) $(dir $(LCSIM_CONDITIONS))

JAVA_OPTS = -Xms1024m -Xmx1024m
CONDITIONS_OPTS=-Dorg.lcsim.cacheDir=$(PWD) -Duser.home=$(PWD)

##### Define geometry targets

$(GEOM_PATH)/compact.xml: $(GEOM_PATH)/dd4hep.xml
	cat $< | sed 's/<includes>.*<\/includes>//;s/type="solenoid"/type="Solenoid"/;s/\<tesla\>/1/g' > $@

$(GEOM_LCDD): $(GEOM_PATH)/compact.xml
	java $(JAVA_OPTS) $(CONDITIONS_OPTS) -jar $(GCONVERTER) -o lcdd $< $@

$(GEOM_GDML): $(GEOM_PATH)/dd4hep.xml
	geoConverter -compact2gdml -input $< -output $@ &> $@.log

$(GEOM_HEPREP): $(GEOM_PATH)/compact.xml
	java $(JAVA_OPTS) $(CONDITIONS_OPTS) -jar $(GCONVERTER) -o heprep $< $@

$(GEOM_PANDORA): $(GEOM_PATH)/compact.xml $$(LCSIM_CONDITIONS)
	java $(JAVA_OPTS) $(CONDITIONS_OPTS) -jar $(GCONVERTER) -o pandora $< $@

%.html: $(GEOM_PATH)/compact.xml $$(LCSIM_CONDITIONS)
	java $(JAVA_OPTS) $(CONDITIONS_OPTS) -jar $(GCONVERTER) -o html $< $@

$(PWD)/.lcsim/cache/$(LCSIM_CONDITIONS_PREFIX_ESCAPED)%.zip: $(GEOM_HEPREP) $(GEOM_LCDD)
	mkdir -p $(@D)
	cd $(GEOM_PATH) && zip -r $@ * &> $@.log

$(GEOM_OVERLAP_CHECK): $(GEOM_GDML) tools/overlapCheck.cpp
	root -b -q -l "tools/overlapCheck.cpp(\"$<\");" | tee $@

##### Define tracking strategy list target

$(STRATEGIES): $(GEOM_PATH)/compact.xml $(GEOM_PATH)/config/prototypeStrategy.xml \
			$(GEOM_PATH)/config/layerWeights.xml $(GEOM_PATH)/config/strategyBuilder.xml \
			$$(LCSIM_CONDITIONS)
	if [ -f $(GEOM_PATH)/config/trainingSample.slcio ]; \
		then java $(JAVA_OPTS) $(CONDITIONS_OPTS) \
			-jar $(CLICSOFT)/distribution/target/lcsim-distribution-*-bin.jar \
			-DprototypeStrategyFile=$(GEOM_PATH)/config/prototypeStrategy.xml \
			-DlayerWeightsFile=$(GEOM_PATH)/config/layerWeights.xml \
			-DtrainingSampleFile=$(GEOM_PATH)/config/trainingSample.slcio \
			-DoutputStrategyFile=$@ \
			$(GEOM_PATH)/config/strategyBuilder.xml \
			&> $@.log; \
	else \
		touch $@; \
	fi

##### Define output targets

# Conversion of promc truth file to slcio
output/%_truth.slcio: input/%.promc
	mkdir -p $(@D)
	java $(JAVA_OPTS) promc2lcio $(abspath $<) $(abspath $@) \
		&> $@.log

# DDSim simulation of truth events
output/%.slcio: output/%_truth.slcio $(GEOM_PATH)/dd4hep.xml nEventsPerRun $(GEOM_PATH)/config/ddsim-steering.py
	time bash -c "time ddsim \
		--runType batch \
		--inputFiles $< \
		--steeringFile $(GEOM_PATH)/config/ddsim-steering.py \
		--compactFile $(GEOM_PATH)/dd4hep.xml \
		--numberOfEvents $(N_EVENTS) \
		--outputFile $@" \
		&> $@.log

# Digitization AND tracking with LCSim
output/%_tracking.slcio: output/%.slcio $(STRATEGIES) \
				$(GEOM_PATH)/config/sid_dbd_prePandora_noOverlay.xml \
				$$(LCSIM_CONDITIONS)
	time bash -c "time java $(JAVA_OPTS) $(CONDITIONS_OPTS) \
		-jar $(CLICSOFT)/distribution/target/lcsim-distribution-*-bin.jar \
		-DinputFile=$< \
		-DtrackingStrategies=$(STRATEGIES) \
		-DoutputFile=$@ \
		$(GEOM_PATH)/config/sid_dbd_prePandora_noOverlay.xml" \
		&> $@.log

# Convert tracking to proio
output/%_tracking.proio.gz: output/%_tracking.slcio
	lcio2proio -o $@ $<

##### Analysis target definitions

%/trackEff.pdf: tools/trackEff.go $(OUTPUT_TRACKING)
	go run tools/trackEff.go -t 40 -o $@ $(shell find $(@D) -name "*_tracking.slcio")

%/trackEff-norm.pdf: tools/trackEff.go $(OUTPUT_TRACKING)
	go run tools/trackEff.go -t 40 -n -o $@ $(shell find $(@D) -name "*_tracking.slcio")

%/trackEff-devAng.pdf: tools/trackEff.go $(OUTPUT_TRACKING)
	go run tools/trackEff.go -t 40 -a -o $@ $(shell find $(@D) -name "*_tracking.slcio")

%/trackEff-pT.pdf: tools/trackEff.go $(OUTPUT_TRACKING)
	go run tools/trackEff.go -t 40 -p -o $@ $(shell find $(@D) -name "*_tracking.slcio")

%/trackEff-pT-norm.pdf: tools/trackEff.go $(OUTPUT_TRACKING)
	go run tools/trackEff.go -t 40 -p -n -o $@ $(shell find $(@D) -name "*_tracking.slcio")

%/clusterDist.pdf: tools/clusterDist.go $(OUTPUT_PANDORA)
	go run tools/clusterDist.go -t 40 -o $@ $(shell find $(@D) -name "*_pandora.slcio")

%/clusterDist-energyWeighted.pdf: tools/clusterDist.go $(OUTPUT_PANDORA)
	go run tools/clusterDist.go -t 40 -e -o $@ $(shell find $(@D) -name "*_pandora.slcio")

%/pfoDist.pdf: tools/PFODist.go $(OUTPUT_PANDORA)
	go run tools/PFODist.go -t 40 -o $@ $(shell find $(@D) -name "*_pandora.slcio")

