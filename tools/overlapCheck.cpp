void overlapCheck(const char *filename) {
    TGeoManager::Import(filename);
    gGeoManager->CheckOverlaps();
    gGeoManager->PrintOverlaps();
}
