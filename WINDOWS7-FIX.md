# Windows 7 Executable - No Rename Needed

## The Solution

The Windows executable is now built as **`mg-advent.exe`** from the start. This avoids the Windows 7 rename delay entirely because:

1. The filename is consistent from first build
2. Windows 7 recognizes it as a known application after first run
3. No renaming means no compatibility re-analysis
4. The embedded manifest declares the original filename as `mg-advent.exe`

## What Was The Problem?

Previously, when renaming executables on Windows 7 32-bit (e.g., from `advent-windows-386.exe` to `advent.exe`), Windows would introduce a 15-20 second delay before the program started on **every launch**.

This happened because Windows 7's Application Experience Service treated renamed executables as "new" applications and performed full compatibility analysis on each launch.

## Root Cause

This delay is caused by Windows 7's **Application Experience Service** and **SmartScreen** performing runtime security and compatibility checks on executables that:

1. Don't have embedded Windows metadata (version info, company name, etc.)
2. Lack an application manifest declaring OS compatibility
3. Are unrecognized by the Windows Application Experience database

When you rename an executable, Windows 7 treats it as a "new" application and performs extensive compatibility analysis, including:

- Checking against the Application Compatibility database
- Analyzing the executable for known compatibility issues
- Validating digital signatures (if present)
- Running heuristic security checks

This process takes 15-20 seconds on Windows 7 systems.

## What's Included

The build scripts now automatically:

1. **Build as `mg-advent.exe`** - Consistent filename from the start
2. **Embed Windows manifest** - Declares Windows 7 compatibility
3. **Include version information** - Company name, product name, version
4. **Set original filename** - Metadata matches actual filename

## For Advanced Troubleshooting

If you still experience delays (rare), see [`WINDOWS7-RENAME-WORKAROUNDS.md`](WINDOWS7-RENAME-WORKAROUNDS.md) for additional solutions like:

- Application compatibility shims
- Antivirus exclusions
- Code signing
- Service configuration

## Files Created

### 1. `cmd/advent/advent.manifest`

XML manifest declaring:

- Windows 7/8/8.1/10 compatibility
- Execution level (asInvoker - no elevation)
- DPI awareness settings

### 2. `cmd/advent/resource.rc`

Windows resource file containing:

- Version information (company, product, copyright)
- Original filename reference
- Embedded manifest inclusion

### 3. Updated Build Scripts

The fix is now **integrated into the standard build scripts**:

- **Linux/Mac:** `scripts/build.sh`
- **Windows:** `scripts/build.bat`

Both scripts automatically detect and use `windres` to embed the manifest when building for Windows.

## How to Build with the Fix

The fix is now **automatically applied** by the standard build scripts. Simply run:

### On Linux or Mac

```bash
# Install MinGW-w64 (includes windres) - OPTIONAL but recommended
# Ubuntu/Debian:
sudo apt-get install mingw-w64

# macOS:
brew install mingw-w64

# Build (automatically embeds manifest if windres is available)
./scripts/build.sh
```

### On Windows

```batch
# Install MinGW-w64 - OPTIONAL but recommended
# Option 1: Download from https://www.mingw-w64.org/
# Option 2: Use Chocolatey
choco install mingw

# Build (automatically embeds manifest if windres is available)
scripts\build.bat
```

**Note:** If `windres` is not installed, the build will still succeed but the executable may experience the 15-20 second delay when renamed on Windows 7.

## What Happens During Build

1. **Automatic Detection**: Build script checks if `windres` is available
2. **Resource Compilation**: If found, `windres` compiles `resource.rc` into `resource.syso`
3. **Go Build**: The Go compiler automatically includes `resource.syso` during build (if present)
4. **Manifest Embedding**: The manifest is embedded in the .exe via the resource file
5. **Cleanup**: `resource.syso` is removed after build
6. **Graceful Fallback**: If `windres` is not found, build continues without manifest (with a warning)

## Verification

After building, you can verify the fix:

### Check for Embedded Manifest (PowerShell)

```powershell
$exe = "dist\advent-windows-386.exe"
$bytes = [System.IO.File]::ReadAllBytes($exe)
$text = [System.Text.Encoding]::ASCII.GetString($bytes)
if ($text -match "supportedOS Id") {
    Write-Host "✓ Manifest embedded successfully" -ForegroundColor Green
} else {
    Write-Host "✗ No manifest found" -ForegroundColor Red
}
```

### Check Version Info (Windows)

Right-click the .exe → Properties → Details tab
You should see:

- File description: "Advent Calendar BBS Door"
- Company: "MisfitGeek BBS"
- Original filename: "advent.exe"

### Test Rename Delay

1. Copy `advent-windows-386.exe` to `test.exe`
2. Run `test.exe` - should start immediately
3. Rename to `anything.exe` - should still start immediately

## Alternative Solutions (if windres is not available)

If you cannot install MinGW-w64/windres, you can:

1. **Use a pre-built executable** with the fix applied
2. **Build on Windows with Visual Studio tools** (rc.exe instead of windres)
3. **Use Go's goversioninfo package** (alternative approach):

   ```bash
   go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo
   ```

## Technical Details

The manifest addresses these Windows 7 checks:

- **Application Compatibility Engine**: Manifest declares OS compatibility, skipping analysis
- **SmartScreen Filter**: Version info provides reputation data
- **UAC Virtualization**: asInvoker level prevents elevation prompts
- **Program Compatibility Assistant**: Explicit compatibility declarations disable PCA checks

## Why This Isn't an Issue on Windows 10+

Windows 10 and later versions:

- Have faster compatibility checking
- Use the SmartScreen cloud service (faster lookups)
- Cache executable metadata more aggressively
- Don't perform as many local heuristic checks

Windows 7's local-only compatibility database is much slower.

## References

- [Application Manifest Documentation](https://docs.microsoft.com/en-us/windows/win32/sbscs/application-manifests)
- [VERSIONINFO Resource](https://docs.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource)
- [Windows Application Experience](https://docs.microsoft.com/en-us/previous-versions/windows/it-pro/windows-7/cc722509(v=ws.10))
