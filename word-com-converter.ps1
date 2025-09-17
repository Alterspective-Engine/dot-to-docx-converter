param(
    [string]$InputFolder = "c:\Users\IgorJericevich\Documents\GitHub\dot-to-docx-converter\MatterSphere Export",
    [string]$OutputFolder = "c:\Users\IgorJericevich\Documents\GitHub\dot-to-docx-converter\MatterSphere Export\Converted"
)

# Ensure output folder exists
if (!(Test-Path $OutputFolder)) {
    New-Item -ItemType Directory -Path $OutputFolder -Force | Out-Null
}

Write-Host "Attempting conversion using Microsoft Word COM automation"
Write-Host "This should handle DOT files more reliably than LibreOffice"
Write-Host ""

try {
    # Create Word application object
    $word = New-Object -ComObject Word.Application
    $word.Visible = $false
    $word.DisplayAlerts = 0  # Disable alerts
    
    Write-Host "✓ Microsoft Word COM object created successfully" -ForegroundColor Green
    
    # Get DOT files
    $dotFiles = Get-ChildItem -Path $InputFolder -Filter "*.dot" -File | Sort-Object Name | Select-Object -First 5  # Test with first 5 files
    $total = $dotFiles.Count
    $converted = 0
    $failed = 0
    
    Write-Host "Testing conversion with first $total files"
    Write-Host ""
    
    foreach ($file in $dotFiles) {
        $outputFile = Join-Path $OutputFolder ($file.BaseName + ".docx")
        
        if (Test-Path $outputFile) {
            Write-Host "Skipping $($file.Name) - already exists"
            $converted++
            continue
        }
        
        Write-Host "Converting $($file.Name)..." -NoNewline
        
        try {
            # Open the DOT file
            $doc = $word.Documents.Open($file.FullName, $false, $true)  # ReadOnly = true
            
            # Save as DOCX
            $doc.SaveAs2($outputFile, 16)  # 16 = wdFormatXMLDocument (DOCX)
            $doc.Close()
            
            if (Test-Path $outputFile) {
                $fileSize = (Get-Item $outputFile).Length
                Write-Host " ✓ Success ($fileSize bytes)" -ForegroundColor Green
                $converted++
            } else {
                Write-Host " ✗ Failed (no output)" -ForegroundColor Red
                $failed++
            }
        }
        catch {
            Write-Host " ✗ Error: $($_.Exception.Message)" -ForegroundColor Red
            $failed++
            
            # Try to close document if it's still open
            try {
                if ($doc) { $doc.Close() }
            } catch {}
        }
    }
    
    # Close Word application
    $word.Quit()
    $word = $null
    
    Write-Host ""
    Write-Host "=== WORD COM CONVERSION RESULTS ===" -ForegroundColor Green
    Write-Host "Converted: $converted/$total files"
    Write-Host "Failed: $failed files"
    
    if ($converted -gt 0) {
        Write-Host ""
        Write-Host "✓ Word COM automation works! You can now run the full conversion." -ForegroundColor Green
        Write-Host "The converted files should open properly in Microsoft Word." -ForegroundColor Green
    }
}
catch {
    Write-Host "✗ Microsoft Word is not available for COM automation" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Falling back to LibreOffice conversion..." -ForegroundColor Yellow
    
    # Fallback to LibreOffice for 5 files
    $dotFiles = Get-ChildItem -Path $InputFolder -Filter "*.dot" -File | Sort-Object Name | Select-Object -First 5
    
    foreach ($file in $dotFiles) {
        $outputFile = Join-Path $OutputFolder ($file.BaseName + "_lo.docx")
        
        if (Test-Path $outputFile) {
            continue
        }
        
        Write-Host "LibreOffice: Converting $($file.Name)..." -NoNewline
        
        $process = Start-Process -FilePath "C:\Program Files\LibreOffice\program\soffice.exe" -ArgumentList @(
            "--headless", "--invisible", "--convert-to", "docx", "--outdir", $OutputFolder, $file.FullName
        ) -Wait -PassThru -WindowStyle Hidden
        
        if ($process.ExitCode -eq 0 -and (Test-Path $outputFile)) {
            Write-Host " Success" -ForegroundColor Green
        } else {
            Write-Host " Failed" -ForegroundColor Red
        }
    }
}

Write-Host ""
Write-Host "Test conversion complete. Check the output folder for results."