<p align="center">
  <img src="./assets/giticon.png" alt="EDxDC Logo">
</p>

<p align="center">
    <img src="https://img.shields.io/badge/-Windows-blue?logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB3aWR0aD0iNjQiIGhlaWdodD0iNjQiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiPgogIDxkZWZzLz4KICA8Zz4KICAgIDxwYXRoIHN0cm9rZT0ibm9uZSIgZmlsbD0iI0ZGRkZGRiIgZD0iTTMxIDM1IFEzMSAzNC4xNSAzMC40NSAzMy42IDI5Ljg1IDMzIDI5IDMzIEwyMyAzMyBRMjIuMTUgMzMgMjEuNiAzMy42IDIxIDM0LjE1IDIxIDM1IEwyMSA0MSBRMjEgNDEuODUgMjEuNiA0Mi40NSAyMi4xNSA0MyAyMyA0MyBMMjkgNDMgUTI5Ljg1IDQzIDMwLjQ1IDQyLjQ1IDMxIDQxLjg1IDMxIDQxIEwzMSAzNSBNMzEgMjMgUTMxIDIyLjE1IDMwLjQ1IDIxLjYgMjkuODUgMjEgMjkgMjEgTDIzIDIxIFEyMi4xNSAyMSAyMS42IDIxLjYgMjEgMjIuMTUgMjEgMjMgTDIxIDI5IFEyMSAyOS44NSAyMS42IDMwLjQ1IDIyLjE1IDMxIDIzIDMxIEwyOSAzMSBRMjkuODUgMzEgMzAuNDUgMzAuNDUgMzEgMjkuODUgMzEgMjkgTDMxIDIzIE0zMyA0MSBRMzMgNDEuODUgMzMuNiA0Mi40NSAzNC4xNSA0MyAzNSA0MyBMNDEgNDMgUTQxLjg1IDQzIDQyLjQ1IDQyLjQ1IDQzIDQxLjg1IDQzIDQxIEw0MyAzNSBRNDMgMzQuMTUgNDIuNDUgMzMuNiA0MS44NSAzMyA0MSAzMyBMMzUgMzMgUTM0LjE1IDMzIDMzLjYgMzMuNiAzMyAzNC4xNSAzMyAzNSBMMzMgNDEgTTQ4IDggUTU2IDggNTYgMTYgTDU2IDQ4IFE1NiA1NiA0OCA1NiBMMTYgNTYgUTggNTYgOCA0OCBMOCAxNiBROCA4IDE2IDggTDQ4IDggTTM1IDMxIEw0MSAzMSBRNDEuODUgMzEgNDIuNDUgMzAuNDUgNDMgMjkuODUgNDMgMjkgTDQzIDIzIFE0MyAyMi4xNSA0Mi40NSAyMS42IDQxLjg1IDIxIDQxIDIxIEwzNSAyMSBRMzQuMTUgMjEgMzMuNiAyMS42IDMzIDIyLjE1IDMzIDIzIEwzMyAyOSBRMzMgMjkuODUgMzMuNiAzMC40NSAzNC4xNSAzMSAzNSAzMSIvPgogIDwvZz4KPC9zdmc%2B" alt="OS: Windows"/>
    <img src="https://img.shields.io/github/license/pellux-network/EDxDC" alt="License"/>
    <img src="https://img.shields.io/github/go-mod/go-version/pellux-network/EDxDC?logo=go&logoSize=auto&label=%20&color=grey" alt="Go Version"/>
    <img src="https://img.shields.io/github/actions/workflow/status/pellux-network/EDxDC/go.yml" alt="Build Status"/>
    <a href="https://github.com/pellux-network/EDxDC/issues">
      <img src="https://img.shields.io/github/issues/pellux-network/EDxDC" alt="GitHub Issues"/>
    </a>
    <a href="https://github.com/pellux-network/EDxDC/pulls">
      <img src="https://img.shields.io/github/issues-pr/pellux-network/EDxDC" alt="GitHub Pull Requests"/>
    </a>
</p>

<p align="center">
  <a href="https://github.com/pellux-network/EDxDC/releases/latest">
    <img src="https://img.shields.io/badge/Download%20Latest%20Release-blue?style=for-the-badge&logo=github" alt="Download Latest Release"/>
  </a>
</p>

# Elite Dangerous E*x*ternal Display Controller

Seamlessly reads Elite Dangerous journal data and presents real-time system, planet, cargo, and other information on your Saitek/Logitech X52 Pro Multi-Function Display.

> [!IMPORTANT] 
> _This software only works with the Saitek/Logitech X52 Pro. The standard X52 HOTAS does not support third-party software for the MFD._

※_Development is ongoing. See the [changelog](https://github.com/pellux-network/EDxDC/blob/master/CHANGELOG.md) for details on recent fixes and features._

## Getting Started

To install EDxDC, you have two options:

1. **Installer (Recommended):**  
   Download and run the latest `EDxDC-vX.X.X-[ReleaseType]-Setup.exe` from the [Releases](https://github.com/pellux-network/EDxDC/releases/latest) page. This will install the application and create shortcuts for easy access.

2. **Portable:**  
   Alternatively, download the latest `EDxDC-vX.X.X-[ReleaseType]-portable-amd64.zip` from the [Releases](https://github.com/pellux-network/EDxDC/releases/latest) page. Unzip it into a location of your choosing such as `C:\Games\`. Then run the included `.exe` directly.

If you haven't modified Elite Dangerous' journal path and don't want to disable any pages, simply run the app and your MFD should immediately begin loading.

If your journal file location is different than the default or you wish to disable any pages, check the [Configuration](https://github.com/pellux-network/EDxDC/wiki/3.-Configuration) page on the [Wiki](https://github.com/pellux-network/EDxDC/wiki) for more details.

> [!TIP] 
> _It is recommended to run a tool that uploads data to the Elite Dangerous Data Network, such as [ED Market Connector](https://github.com/Marginal/EDMarketConnector). Doing this will ensure that any new discoveries can be shown on the display._

## [Wiki](https://github.com/pellux-network/EDxDC/wiki)

For additional information or help with any encountered issues, visit the [Wiki](https://github.com/pellux-network/EDxDC/wiki). This includes info on getting the correct drivers for the X52

## Credits

- Huge thanks to [pbxx](https://github.com/pbxx) for icons, page layouts, and huge general improvements to the codebase as well as helping clean up the original code.

## Attribution

- This software was originally based off of [EDx52display](https://github.com/peterbn/EDx52display) by [Peter Pakkenberg](https://github.com/peterbn)

<p style="font-size: 12px" align="right">
  <a href="#EDxDC">Jump to top</a>
</p>
