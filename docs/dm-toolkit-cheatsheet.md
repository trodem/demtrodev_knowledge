# dm toolkit Cheat Sheet

## 1) Open the wizard

```powershell
dm toolkit
```

## 2) Create a new toolkit

```powershell
dm toolkit new --name MSWord --prefix word --category office
```

Creates:

`plugins/functions/office/MSWord_Toolkit.ps1`

## 3) Add a new function

```powershell
dm toolkit add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func export_pdf --param InputPath --param OutputPath --confirm
```

## 4) Validate toolkits

```powershell
dm toolkit validate
```

## 5) Useful variants

```powershell
# ensure shared helper exists in plugins/utils.ps1
dm toolkit add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func open --require-helper _assert_path_exists --param Path
```

```powershell
# ensure shared variable exists in plugins/variables.ps1
dm toolkit add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func export_default --require-var DM_WORD_TEMPLATE=normal.dotm
```
