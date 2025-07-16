[Setup]
AppName=EDxDC
AppId=Pellux-Network.EDxDC
AppVersion=1.1.0-beta
DefaultDirName={code:GetInstallDir}
DefaultGroupName=EDxDC
OutputDir=.
OutputBaseFilename=EDxDC-v1.1.0-beta-Setup
Compression=lzma
SolidCompression=yes
WizardStyle=modern
SetupIconFile=icon-install.ico
UninstallDisplayIcon=icon-uninstall.ico
WizardImageFile=banner-install-complete.bmp
WizardSmallImageFile=icon-installing.bmp
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
ArchitecturesAllowed=win64
ArchitecturesInstallIn64BitMode=win64


[Files]
// Update the version and file names as needed
Source: "EDxDC.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "names\*"; DestDir: "{app}\names"; Flags: ignoreversion recursesubdirs
Source: "bin\*"; DestDir: "{app}\bin"; Flags: ignoreversion recursesubdirs

[Tasks]
Name: "desktopicon"; Description: "Create a &desktop icon"; GroupDescription: "Additional icons:"; Flags: unchecked;
Name: "startmenuicon"; Description: "Create a &Start Menu icon"; GroupDescription: "Additional icons:";

[Icons]
Name: "{group}\EDxDC"; Filename: "{app}\EDxDC.exe"; Tasks: startmenuicon
Name: "{group}\Uninstall EDxDC"; Filename: "{uninstallexe}"; Tasks: startmenuicon
Name: "{userdesktop}\EDxDC"; Filename: "{app}\EDxDC.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\EDxDC.exe"; Description: "Launch EDxDC"; Flags: nowait postinstall skipifsilent

[Code]
function GetInstallDir(Default: String): String;
begin
  if IsAdminInstallMode then
    Result := ExpandConstant('{autopf}\EDxDC')
  else
    Result := ExpandConstant('{localappdata}\EDxDC');
end;


var
  RetainLogs: Boolean;
  RetainConf: Boolean;

procedure InitializeUninstallProgressForm;
var
  UserChoice: Integer;
begin
  UserChoice := MsgBox('Do you want to keep the logs folder and main.conf?', mbConfirmation, MB_YESNOCANCEL);
  if UserChoice = IDYES then begin
    RetainLogs := True;
    RetainConf := True;
  end else begin
    RetainLogs := False;
    RetainConf := False;
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then begin
    if not RetainLogs then
      DelTree(ExpandConstant('{userappdata}\EDxDC\logs'), True, True, True);
    if not RetainConf then
      DeleteFile(ExpandConstant('{userappdata}\EDxDC\main.conf'));
    // If neither logs nor conf are retained, remove the whole directory
    if (not RetainLogs) and (not RetainConf) then
      DelTree(ExpandConstant('{userappdata}\EDxDC'), True, True, True);
  end;
end;