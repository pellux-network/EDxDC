# If anyone sees this and wonders why I'm using individual PNGs instead of a single SVG file,
# it's because the "-define icon:auto-resize=48,32,24,16" option in magick convert does not 
# seem to work with multiple svg files for different sizes.
# 
# If you know a way to do this, please let me know.

param(
    [switch]$OnlyGit
)

# If $OnlyGit is true, only convert the SVG to PNG for the README.md
if ($OnlyGit) {
    magick convert -background none icon.svg -resize 192x192 .\giticon.png
    Write-Host "Conversion complete. The PNG file for README.md is located at .\giticon.png"
    exit
} else {

    # create a new directory for the png files
    $PngDir = ".\temp"
    $PngDirExists = Test-Path -Path $PngDir

    if (!$PngDirExists) {
        New-Item -ItemType Directory -Path $PngDir > $null
    }
    else {
        Remove-Item -Path $PngDir -Recurse -Force
        New-Item -ItemType Directory -Path $PngDir > $null
    }
    # convert the svg file to png files of different sizes
    magick convert -background none icon.svg -resize 256x256 $PngDir/icon-256.png
    magick convert -background none icon.svg -resize 128x128 $PngDir/icon-128.png
    magick convert -background none icon.svg -resize 64x64 $PngDir/icon-64.png
    magick convert -background none icon.svg -resize 48x48 $PngDir/icon-48.png
    magick convert -background none icon-small.svg -resize 32x32 $PngDir/icon-32.png
    magick convert -background none icon-small.svg -resize 24x24 $PngDir/icon-24.png
    magick convert -background none icon-small.svg -resize 16x16 $PngDir/icon-16.png

    # convert the png files to an ico file
    magick convert -background transparent $PngDir/icon-256.png $PngDir/icon-128.png $PngDir/icon-64.png $PngDir/icon-48.png -background transparent $PngDir/icon-32.png -background transparent $PngDir/icon-24.png $PngDir/icon-16.png -background transparent ..\icon.ico

    # delete the png directory and its contents
    Remove-Item -Path $PngDir -Recurse -Force

    # convert the svg file to a png file for use in the README.md
    magick convert -background none icon.svg -resize 192x192 .\giticon.png

    Write-Host "Conversion complete. The ICO file is located at ..\icon.ico"
}