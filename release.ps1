$version = Read-Host "Enter the release version"

go clean

Remove-Item  .\EDxDC* -Force -Recurse -ErrorAction SilentlyContinue

mkdir EDxDC-$version

go build -ldflags "-H=windowsgui -s -w" -o EDxDC-$version.exe # Strip debug info

Copy-Item -Path EDxDC-$version.exe,conf.yaml,LICENSE,README.md,names,bin -Destination .\EDxDC-$version -Recurse

7z.exe a EDxDC-$version-portable-amd64.zip .\EDxDC-$version

pause