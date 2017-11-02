from DDSim.DD4hepSimulation import DD4hepSimulation
from SystemOfUnits import mm, GeV, MeV, keV

SIM = DD4hepSimulation()

SIM.action.tracker = "Geant4TrackerAction"
SIM.enableDetailedShowerMode = True
SIM.crossingAngleBoost = 0.007
SIM.field.min_chord_step = 0.01*mm
SIM.field.eps_min = -1
SIM.field.eps_max = -1
SIM.field.delta_chord = -1
SIM.field.delta_intersection = -1
SIM.field.delta_one_step = -1
SIM.field.largest_step = -1
SIM.field.stepper = "ClassicalRK4"
