# This script converts the SVG mock screen to PNG for use in the README.md

param(
    [Parameter(Mandatory = $true)]
    [string]$Name,
    [switch]$Diagram
)

$MockDir = ".\mockups"
$MockDirExists = Test-Path -Path $MockDir

# Create the directory if it does not exist
if (!$MockDirExists) {
    New-Item -ItemType Directory -Path $MockDir > $null
}

if ($Diagram) {
    # If the Diagram switch is set, use the diagram SVG
    $SvgFile = "./mockdiagram.svg"
} else {
    # Otherwise, use the regular mock screen SVG
    $SvgFile = "./mockscreen.svg"
}
inkscape $SvgFile -o "./mockups/$Name.png"

Write-Host "Conversion complete. The PNG mockup is located at ./mockups/$Name.png"