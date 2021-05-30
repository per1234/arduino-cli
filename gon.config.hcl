source = ["dist/arduino-cli_osx_darwin_amd64/arduino-cli"]
bundle_id = "cc.arduino.arduino-cli"

sign {
  application_identity = "Developer ID Application: Per Tillisch (9M5NQMNWBJ)"
}

# Ask Gon for zip output to force notarization process to take place.
# The CI will ignore the zip output, using the signed binary only.
zip {
  output_path = "arduino-cli.zip"
}
