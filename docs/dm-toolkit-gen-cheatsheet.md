# dm-toolkit-gen Cheat Sheet

## 1) Use Built-in `dm toolkit` (recommended)

```powershell
dm toolkit
```

```powershell
dm toolkit new --name MSWord --prefix word --category office
```

```powershell
dm toolkit add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func export_pdf --param InputPath --param OutputPath --confirm
```

```powershell
dm toolkit validate
```

## 2) Create a New Toolkit

```powershell
dm toolkit new --name MSWord --prefix word --category office
```

Creates:

`plugins/functions/office/MSWord_Toolkit.ps1`

## 3) Add a New Function

```powershell
dm toolkit add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func export_pdf --param InputPath --param OutputPath --confirm
```

## 4) Validate Toolkits

```powershell
dm toolkit validate
```

## 5) Useful Variants

```powershell
# ensure shared helper exists in plugins/utils.ps1
dm toolkit add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func open --require-helper _assert_path_exists --param Path
```

```powershell
# ensure shared variable exists in plugins/variables.ps1
dm toolkit add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func export_default --require-var DM_WORD_TEMPLATE=normal.dotm
```

## 6) Optional standalone mode (if needed)

```powershell
go build -o dist/dm-toolkit-gen.exe ./cmd/dm-toolkit-gen
.\dist\dm-toolkit-gen.exe init --repo . --name MSWord --prefix word --category office
```
