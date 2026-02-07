#!/usr/bin/env pwsh

# Test all /list endpoints in Espyna API
# Usage: ./test-all-list-endpoints.ps1
# Usage with prefix filter: 
#   ./test-all-list-endpoints.ps1 -Prefix "/api/framework"
#   ./test-all-list-endpoints.ps1 -Prefix "/api/framework/framework"
#   ./test-all-list-endpoints.ps1 -Prefix "/api/entity"
# Usage with timestamped results: 
#   PowerShell: $timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'; powershell.exe scripts/test/test-all-list-endpoints.ps1 | Tee-Object -FilePath "tests/e2e/results/$timestamp-api-test-results.txt"
#   Bash:       timestamp=$(date +%Y%m%d-%H%M%S); powershell.exe scripts/test/test-all-list-endpoints.ps1 | tee "tests/e2e/results/$timestamp-api-test-results.txt"

param(
    [string]$ResultsFile = $null,
    [string]$Prefix = $null
)

# Initialize test results collection
$TestResults = @()
$body = "{}"

# Function to capture Write-Host output
function Write-HostWithCapture {
    param(
        [string]$Message,
        [ConsoleColor]$ForegroundColor = "White"
    )
    Write-Host $Message -ForegroundColor $ForegroundColor
    $script:TestResults += $Message + "`n"
}

# Generate timestamp if results file not specified
if (-not $ResultsFile) {
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $ResultsFile = "tests/e2e/results/$timestamp-api-test-results.txt"
    Write-HostWithCapture "Results will be saved to: $ResultsFile" -ForegroundColor Yellow
}

$baseUrl = "http://localhost:8080"
$headers = @{
    "Content-Type" = "application/json"
    "X-Leapfor-MockBusinessType" = "education"
}

# Function to filter endpoints by prefix
function Filter-Endpoints {
    param(
        [array]$Endpoints,
        [string]$Prefix
    )
    
    if ($Prefix) {
        return $Endpoints | Where-Object { $_ -like "$Prefix*" }
    } else {
        return $Endpoints
    }
}

if ($Prefix) {
    Write-HostWithCapture "Testing endpoints with prefix: $Prefix" -ForegroundColor Green
    Write-HostWithCapture "=============================================" -ForegroundColor Green
} else {
    Write-HostWithCapture "Testing all /list endpoints in Espyna API (Fiber Framework)" -ForegroundColor Green
    Write-HostWithCapture "=============================================" -ForegroundColor Green
}

# Entity Domain Endpoints
$entityEndpoints = @(
    "/api/entity/admin/list",
    "/api/entity/client/list", 
    "/api/entity/client-attribute/list",
    "/api/entity/delegate/list",
    "/api/entity/delegate-client/list",
    "/api/entity/group/list",
    "/api/entity/location/list",
    "/api/entity/location-attribute/list",
    "/api/entity/manager/list",
    "/api/entity/permission/list",
    "/api/entity/role/list",
    "/api/entity/role-permission/list",
    "/api/entity/staff/list",
    "/api/entity/user/list",
    "/api/entity/workspace/list",
    "/api/entity/workspace-user/list",
    "/api/entity/workspace-user-role/list"
)

$filteredEntityEndpoints = Filter-Endpoints -Endpoints $entityEndpoints -Prefix $Prefix

if ($filteredEntityEndpoints.Count -gt 0) {
    Write-HostWithCapture "`nENTITY DOMAIN" -ForegroundColor Yellow

foreach ($endpoint in $filteredEntityEndpoints) {
    Write-HostWithCapture "Testing: $endpoint" -ForegroundColor Cyan
    try {
        $response = Invoke-RestMethod -Uri "$baseUrl$endpoint" -Method POST -Headers $headers -Body $body
        if ($response.success -eq $true) {
            $dataCount = if ($response.data) { $response.data.Count } else { 0 }
            Write-HostWithCapture "  ✅ SUCCESS - $dataCount items returned" -ForegroundColor Green
            
            # Test read endpoint if we have data
            if ($dataCount -gt 0 -and $response.data) {
                $firstItem = $response.data[0]
                if ($firstItem.id) {
                    $readEndpoint = $endpoint -replace "/list", "/read"
                    Write-HostWithCapture "    Testing read endpoint: $readEndpoint" -ForegroundColor DarkCyan
                    try {
                        $readBody = @{ data = @{ id = $firstItem.id } } | ConvertTo-Json -Depth 3
                        $readResponse = Invoke-RestMethod -Uri "$baseUrl$readEndpoint" -Method POST -Headers $headers -Body $readBody
                        if ($readResponse.success -eq $true) {
                            $readDataCount = if ($readResponse.data) { $readResponse.data.Count } else { 0 }
                            Write-HostWithCapture "    ✅ READ SUCCESS - $readDataCount items returned" -ForegroundColor DarkGreen
                            
                            # Test create endpoint using read data with new ID
                            if ($readResponse.data -and $readResponse.data.Count -gt 0) {
                                $createEndpoint = $endpoint -replace "/list", "/create"
                                Write-HostWithCapture "    Testing create endpoint: $createEndpoint" -ForegroundColor DarkMagenta
                                try {
                                    $createData = $readResponse.data[0] | ConvertTo-Json -Depth 10 | ConvertFrom-Json
                                    # Generate new ID for create operation
                                    $createData.id = [System.Guid]::NewGuid().ToString()
                                    if ($createData.name) { $createData.name += "_test_copy" }
                                    if ($createData.title) { $createData.title += "_test_copy" }
                                    
                                    $createBody = @{ data = $createData } | ConvertTo-Json -Depth 10
                                    $createResponse = Invoke-RestMethod -Uri "$baseUrl$createEndpoint" -Method POST -Headers $headers -Body $createBody
                                    if ($createResponse.success -eq $true) {
                                        Write-HostWithCapture "    ✅ CREATE SUCCESS - New item created" -ForegroundColor DarkGreen
                                        
                                        # Test delete endpoint using the created item
                                        $deleteEndpoint = $endpoint -replace "/list", "/delete"
                                        Write-HostWithCapture "    Testing delete endpoint: $deleteEndpoint" -ForegroundColor DarkRed
                                        try {
                                            $deleteBody = @{ data = @{ id = $createResponse.data[0].id } } | ConvertTo-Json -Depth 3
                                            
                                            # Use Invoke-WebRequest for better error handling
                                            try {
                                                $webResponse = Invoke-WebRequest -Uri "$baseUrl$deleteEndpoint" -Method POST -Headers $headers -Body $deleteBody -ContentType "application/json"
                                                $deleteResponse = $webResponse.Content | ConvertFrom-Json
                                                
                                                if ($deleteResponse.success -eq $true) {
                                                    Write-HostWithCapture "    ✅ DELETE SUCCESS - Item deleted" -ForegroundColor DarkGreen
                                                } else {
                                                    $errorMsg = if ($deleteResponse.message) { $deleteResponse.message } else { "success=false" }
                                                    Write-HostWithCapture "    ❌ DELETE FAILED - $errorMsg" -ForegroundColor Red
                                                }
                                            } catch {
                                                # Handle HTTP errors with detailed response
                                                $errorMessage = $_.Exception.Message
                                                
                                                if ($_.Exception.Response) {
                                                    try {
                                                        $responseStream = $_.Exception.Response.GetResponseStream()
                                                        $reader = New-Object System.IO.StreamReader($responseStream)
                                                        $responseBody = $reader.ReadToEnd()
                                                        $reader.Close()
                                                        $responseStream.Close()
                                                        
                                                        $errorJson = $responseBody | ConvertFrom-Json -ErrorAction SilentlyContinue
                                                        if ($errorJson -and $errorJson.message) {
                                                            $errorMessage += " | API Error: $($errorJson.message)"
                                                        } elseif ($errorJson -and $errorJson.error) {
                                                            $errorMessage += " | API Error: $($errorJson.error)"
                                                        } elseif ($responseBody -and $responseBody.Length -gt 0) {
                                                            $truncatedBody = if ($responseBody.Length -gt 200) { $responseBody.Substring(0, 200) + "..." } else { $responseBody }
                                                            $errorMessage += " | Response: $truncatedBody"
                                                        }
                                                    } catch {
                                                        # Ignore errors in error response parsing
                                                    }
                                                }
                                                
                                                Write-HostWithCapture "    ❌ DELETE ERROR - $errorMessage" -ForegroundColor Red
                                            }
                                        } catch {
                                            Write-HostWithCapture "    ❌ DELETE ERROR - Unexpected error: $($_.Exception.Message)" -ForegroundColor Red
                                        }
                                    } else {
                                        Write-HostWithCapture "    ❌ CREATE FAILED - success=false" -ForegroundColor Red
                                    }
                                } catch {
                                    Write-HostWithCapture "    ❌ CREATE ERROR - $($_.Exception.Message)" -ForegroundColor Red
                                }
                            }
                        } else {
                            Write-HostWithCapture "    ❌ READ FAILED - success=false" -ForegroundColor Red
                        }
                    } catch {
                        Write-HostWithCapture "    ❌ READ ERROR - $($_.Exception.Message)" -ForegroundColor Red
                    }
                }
            }
        } else {
            Write-HostWithCapture "  ❌ FAILED - success=false" -ForegroundColor Red
        }
    } catch {
        Write-HostWithCapture "  ❌ ERROR - $($_.Exception.Message)" -ForegroundColor Red
    }
}
}

# Event Domain Endpoints
$eventEndpoints = @(
    "/api/event/event/list"
)

$filteredEventEndpoints = Filter-Endpoints -Endpoints $eventEndpoints -Prefix $Prefix

if ($filteredEventEndpoints.Count -gt 0) {
    Write-HostWithCapture "`nEVENT DOMAIN" -ForegroundColor Yellow

foreach ($endpoint in $filteredEventEndpoints) {
    Write-HostWithCapture "Testing: $endpoint" -ForegroundColor Cyan
    try {
        $response = Invoke-RestMethod -Uri "$baseUrl$endpoint" -Method POST -Headers $headers -Body $body
        if ($response.success -eq $true) {
            $dataCount = if ($response.data) { $response.data.Count } else { 0 }
            Write-HostWithCapture "  ✅ SUCCESS - $dataCount items returned" -ForegroundColor Green
            
            # Test read endpoint if we have data
            if ($dataCount -gt 0 -and $response.data) {
                $firstItem = $response.data[0]
                if ($firstItem.id) {
                    $readEndpoint = $endpoint -replace "/list", "/read"
                    Write-HostWithCapture "    Testing read endpoint: $readEndpoint" -ForegroundColor DarkCyan
                    try {
                        $readBody = @{ data = @{ id = $firstItem.id } } | ConvertTo-Json -Depth 3
                        $readResponse = Invoke-RestMethod -Uri "$baseUrl$readEndpoint" -Method POST -Headers $headers -Body $readBody
                        if ($readResponse.success -eq $true) {
                            $readDataCount = if ($readResponse.data) { $readResponse.data.Count } else { 0 }
                            Write-HostWithCapture "    ✅ READ SUCCESS - $readDataCount items returned" -ForegroundColor DarkGreen
                            
                            # Test create endpoint using read data with new ID
                            if ($readResponse.data -and $readResponse.data.Count -gt 0) {
                                $createEndpoint = $endpoint -replace "/list", "/create"
                                Write-HostWithCapture "    Testing create endpoint: $createEndpoint" -ForegroundColor DarkMagenta
                                try {
                                    $createData = $readResponse.data[0] | ConvertTo-Json -Depth 10 | ConvertFrom-Json
                                    # Generate new ID for create operation
                                    $createData.id = [System.Guid]::NewGuid().ToString()
                                    if ($createData.name) { $createData.name += "_test_copy" }
                                    if ($createData.title) { $createData.title += "_test_copy" }
                                    
                                    $createBody = @{ data = $createData } | ConvertTo-Json -Depth 10
                                    $createResponse = Invoke-RestMethod -Uri "$baseUrl$createEndpoint" -Method POST -Headers $headers -Body $createBody
                                    if ($createResponse.success -eq $true) {
                                        Write-HostWithCapture "    ✅ CREATE SUCCESS - New item created" -ForegroundColor DarkGreen
                                        
                                        # Test delete endpoint using the created item
                                        $deleteEndpoint = $endpoint -replace "/list", "/delete"
                                        Write-HostWithCapture "    Testing delete endpoint: $deleteEndpoint" -ForegroundColor DarkRed
                                        try {
                                            $deleteBody = @{ data = @{ id = $createResponse.data[0].id } } | ConvertTo-Json -Depth 3
                                            
                                            # Use Invoke-WebRequest for better error handling
                                            try {
                                                $webResponse = Invoke-WebRequest -Uri "$baseUrl$deleteEndpoint" -Method POST -Headers $headers -Body $deleteBody -ContentType "application/json"
                                                $deleteResponse = $webResponse.Content | ConvertFrom-Json
                                                
                                                if ($deleteResponse.success -eq $true) {
                                                    Write-HostWithCapture "    ✅ DELETE SUCCESS - Item deleted" -ForegroundColor DarkGreen
                                                } else {
                                                    $errorMsg = if ($deleteResponse.message) { $deleteResponse.message } else { "success=false" }
                                                    Write-HostWithCapture "    ❌ DELETE FAILED - $errorMsg" -ForegroundColor Red
                                                }
                                            } catch {
                                                # Handle HTTP errors with detailed response
                                                $errorMessage = $_.Exception.Message
                                                
                                                if ($_.Exception.Response) {
                                                    try {
                                                        $responseStream = $_.Exception.Response.GetResponseStream()
                                                        $reader = New-Object System.IO.StreamReader($responseStream)
                                                        $responseBody = $reader.ReadToEnd()
                                                        $reader.Close()
                                                        $responseStream.Close()
                                                        
                                                        $errorJson = $responseBody | ConvertFrom-Json -ErrorAction SilentlyContinue
                                                        if ($errorJson -and $errorJson.message) {
                                                            $errorMessage += " | API Error: $($errorJson.message)"
                                                        } elseif ($errorJson -and $errorJson.error) {
                                                            $errorMessage += " | API Error: $($errorJson.error)"
                                                        } elseif ($responseBody -and $responseBody.Length -gt 0) {
                                                            $truncatedBody = if ($responseBody.Length -gt 200) { $responseBody.Substring(0, 200) + "..." } else { $responseBody }
                                                            $errorMessage += " | Response: $truncatedBody"
                                                        }
                                                    } catch {
                                                        # Ignore errors in error response parsing
                                                    }
                                                }
                                                
                                                Write-HostWithCapture "    ❌ DELETE ERROR - $errorMessage" -ForegroundColor Red
                                            }
                                        } catch {
                                            Write-HostWithCapture "    ❌ DELETE ERROR - Unexpected error: $($_.Exception.Message)" -ForegroundColor Red
                                        }
                                    } else {
                                        Write-HostWithCapture "    ❌ CREATE FAILED - success=false" -ForegroundColor Red
                                    }
                                } catch {
                                    Write-HostWithCapture "    ❌ CREATE ERROR - $($_.Exception.Message)" -ForegroundColor Red
                                }
                            }
                        } else {
                            Write-HostWithCapture "    ❌ READ FAILED - success=false" -ForegroundColor Red
                        }
                    } catch {
                        Write-HostWithCapture "    ❌ READ ERROR - $($_.Exception.Message)" -ForegroundColor Red
                    }
                }
            }
        } else {
            Write-HostWithCapture "  ❌ FAILED - success=false" -ForegroundColor Red
        }
    } catch {
        Write-HostWithCapture "  ❌ ERROR - $($_.Exception.Message)" -ForegroundColor Red
    }
}
}

# Framework Domain Endpoints
$frameworkEndpoints = @(
    "/api/framework/framework/list",
    "/api/framework/objective/list",
    "/api/framework/task/list"
)

$filteredFrameworkEndpoints = Filter-Endpoints -Endpoints $frameworkEndpoints -Prefix $Prefix

if ($filteredFrameworkEndpoints.Count -gt 0) {
    Write-HostWithCapture "`nFRAMEWORK DOMAIN" -ForegroundColor Yellow

foreach ($endpoint in $filteredFrameworkEndpoints) {
    Write-HostWithCapture "Testing: $endpoint" -ForegroundColor Cyan
    try {
        $response = Invoke-RestMethod -Uri "$baseUrl$endpoint" -Method POST -Headers $headers -Body $body
        if ($response.success -eq $true) {
            $dataCount = if ($response.data) { $response.data.Count } else { 0 }
            Write-HostWithCapture "  ✅ SUCCESS - $dataCount items returned" -ForegroundColor Green
            
            # Test read endpoint if we have data
            if ($dataCount -gt 0 -and $response.data) {
                $firstItem = $response.data[0]
                if ($firstItem.id) {
                    $readEndpoint = $endpoint -replace "/list", "/read"
                    Write-HostWithCapture "    Testing read endpoint: $readEndpoint" -ForegroundColor DarkCyan
                    try {
                        $readBody = @{ data = @{ id = $firstItem.id } } | ConvertTo-Json -Depth 3
                        $readResponse = Invoke-RestMethod -Uri "$baseUrl$readEndpoint" -Method POST -Headers $headers -Body $readBody
                        if ($readResponse.success -eq $true) {
                            $readDataCount = if ($readResponse.data) { $readResponse.data.Count } else { 0 }
                            Write-HostWithCapture "    ✅ READ SUCCESS - $readDataCount items returned" -ForegroundColor DarkGreen
                            
                            # Test create endpoint using read data with new ID
                            if ($readResponse.data -and $readResponse.data.Count -gt 0) {
                                $createEndpoint = $endpoint -replace "/list", "/create"
                                Write-HostWithCapture "    Testing create endpoint: $createEndpoint" -ForegroundColor DarkMagenta
                                try {
                                    $createData = $readResponse.data[0] | ConvertTo-Json -Depth 10 | ConvertFrom-Json
                                    # Generate new ID for create operation
                                    $createData.id = [System.Guid]::NewGuid().ToString()
                                    if ($createData.name) { $createData.name += "_test_copy" }
                                    if ($createData.title) { $createData.title += "_test_copy" }
                                    
                                    $createBody = @{ data = $createData } | ConvertTo-Json -Depth 10
                                    $createResponse = Invoke-RestMethod -Uri "$baseUrl$createEndpoint" -Method POST -Headers $headers -Body $createBody
                                    if ($createResponse.success -eq $true) {
                                        Write-HostWithCapture "    ✅ CREATE SUCCESS - New item created" -ForegroundColor DarkGreen
                                        
                                        # Test delete endpoint using the created item
                                        $deleteEndpoint = $endpoint -replace "/list", "/delete"
                                        Write-HostWithCapture "    Testing delete endpoint: $deleteEndpoint" -ForegroundColor DarkRed
                                        try {
                                            $deleteBody = @{ data = @{ id = $createResponse.data[0].id } } | ConvertTo-Json -Depth 3
                                            
                                            # Use Invoke-WebRequest for better error handling
                                            try {
                                                $webResponse = Invoke-WebRequest -Uri "$baseUrl$deleteEndpoint" -Method POST -Headers $headers -Body $deleteBody -ContentType "application/json"
                                                $deleteResponse = $webResponse.Content | ConvertFrom-Json
                                                
                                                if ($deleteResponse.success -eq $true) {
                                                    Write-HostWithCapture "    ✅ DELETE SUCCESS - Item deleted" -ForegroundColor DarkGreen
                                                } else {
                                                    $errorMsg = if ($deleteResponse.message) { $deleteResponse.message } else { "success=false" }
                                                    Write-HostWithCapture "    ❌ DELETE FAILED - $errorMsg" -ForegroundColor Red
                                                }
                                            } catch {
                                                # Handle HTTP errors with detailed response
                                                $errorMessage = $_.Exception.Message
                                                
                                                if ($_.Exception.Response) {
                                                    try {
                                                        $responseStream = $_.Exception.Response.GetResponseStream()
                                                        $reader = New-Object System.IO.StreamReader($responseStream)
                                                        $responseBody = $reader.ReadToEnd()
                                                        $reader.Close()
                                                        $responseStream.Close()
                                                        
                                                        $errorJson = $responseBody | ConvertFrom-Json -ErrorAction SilentlyContinue
                                                        if ($errorJson -and $errorJson.message) {
                                                            $errorMessage += " | API Error: $($errorJson.message)"
                                                        } elseif ($errorJson -and $errorJson.error) {
                                                            $errorMessage += " | API Error: $($errorJson.error)"
                                                        } elseif ($responseBody -and $responseBody.Length -gt 0) {
                                                            $truncatedBody = if ($responseBody.Length -gt 200) { $responseBody.Substring(0, 200) + "..." } else { $responseBody }
                                                            $errorMessage += " | Response: $truncatedBody"
                                                        }
                                                    } catch {
                                                        # Ignore errors in error response parsing
                                                    }
                                                }
                                                
                                                Write-HostWithCapture "    ❌ DELETE ERROR - $errorMessage" -ForegroundColor Red
                                            }
                                        } catch {
                                            Write-HostWithCapture "    ❌ DELETE ERROR - Unexpected error: $($_.Exception.Message)" -ForegroundColor Red
                                        }
                                    } else {
                                        Write-HostWithCapture "    ❌ CREATE FAILED - success=false" -ForegroundColor Red
                                    }
                                } catch {
                                    Write-HostWithCapture "    ❌ CREATE ERROR - $($_.Exception.Message)" -ForegroundColor Red
                                }
                            }
                        } else {
                            Write-HostWithCapture "    ❌ READ FAILED - success=false" -ForegroundColor Red
                        }
                    } catch {
                        Write-HostWithCapture "    ❌ READ ERROR - $($_.Exception.Message)" -ForegroundColor Red
                    }
                }
            }
        } else {
            Write-HostWithCapture "  ❌ FAILED - success=false" -ForegroundColor Red
        }
    } catch {
        Write-HostWithCapture "  ❌ ERROR - $($_.Exception.Message)" -ForegroundColor Red
    }
}
}

# Payment Domain Endpoints
$paymentEndpoints = @(
    "/api/payment/payment/list",
    "/api/payment/payment-method/list",
    "/api/payment/payment-profile/list"
)

$filteredPaymentEndpoints = Filter-Endpoints -Endpoints $paymentEndpoints -Prefix $Prefix

if ($filteredPaymentEndpoints.Count -gt 0) {
    Write-HostWithCapture "`nPAYMENT DOMAIN" -ForegroundColor Yellow

foreach ($endpoint in $filteredPaymentEndpoints) {
    Write-HostWithCapture "Testing: $endpoint" -ForegroundColor Cyan
    try {
        $response = Invoke-RestMethod -Uri "$baseUrl$endpoint" -Method POST -Headers $headers -Body $body
        if ($response.success -eq $true) {
            $dataCount = if ($response.data) { $response.data.Count } else { 0 }
            Write-HostWithCapture "  ✅ SUCCESS - $dataCount items returned" -ForegroundColor Green
            
            # Test read endpoint if we have data
            if ($dataCount -gt 0 -and $response.data) {
                $firstItem = $response.data[0]
                if ($firstItem.id) {
                    $readEndpoint = $endpoint -replace "/list", "/read"
                    Write-HostWithCapture "    Testing read endpoint: $readEndpoint" -ForegroundColor DarkCyan
                    try {
                        $readBody = @{ data = @{ id = $firstItem.id } } | ConvertTo-Json -Depth 3
                        $readResponse = Invoke-RestMethod -Uri "$baseUrl$readEndpoint" -Method POST -Headers $headers -Body $readBody
                        if ($readResponse.success -eq $true) {
                            $readDataCount = if ($readResponse.data) { $readResponse.data.Count } else { 0 }
                            Write-HostWithCapture "    ✅ READ SUCCESS - $readDataCount items returned" -ForegroundColor DarkGreen
                            
                            # Test create endpoint using read data with new ID
                            if ($readResponse.data -and $readResponse.data.Count -gt 0) {
                                $createEndpoint = $endpoint -replace "/list", "/create"
                                Write-HostWithCapture "    Testing create endpoint: $createEndpoint" -ForegroundColor DarkMagenta
                                try {
                                    $createData = $readResponse.data[0] | ConvertTo-Json -Depth 10 | ConvertFrom-Json
                                    # Generate new ID for create operation
                                    $createData.id = [System.Guid]::NewGuid().ToString()
                                    if ($createData.name) { $createData.name += "_test_copy" }
                                    if ($createData.title) { $createData.title += "_test_copy" }
                                    
                                    $createBody = @{ data = $createData } | ConvertTo-Json -Depth 10
                                    $createResponse = Invoke-RestMethod -Uri "$baseUrl$createEndpoint" -Method POST -Headers $headers -Body $createBody
                                    if ($createResponse.success -eq $true) {
                                        Write-HostWithCapture "    ✅ CREATE SUCCESS - New item created" -ForegroundColor DarkGreen
                                        
                                        # Test delete endpoint using the created item
                                        $deleteEndpoint = $endpoint -replace "/list", "/delete"
                                        Write-HostWithCapture "    Testing delete endpoint: $deleteEndpoint" -ForegroundColor DarkRed
                                        try {
                                            $deleteBody = @{ data = @{ id = $createResponse.data[0].id } } | ConvertTo-Json -Depth 3
                                            
                                            # Use Invoke-WebRequest for better error handling
                                            try {
                                                $webResponse = Invoke-WebRequest -Uri "$baseUrl$deleteEndpoint" -Method POST -Headers $headers -Body $deleteBody -ContentType "application/json"
                                                $deleteResponse = $webResponse.Content | ConvertFrom-Json
                                                
                                                if ($deleteResponse.success -eq $true) {
                                                    Write-HostWithCapture "    ✅ DELETE SUCCESS - Item deleted" -ForegroundColor DarkGreen
                                                } else {
                                                    $errorMsg = if ($deleteResponse.message) { $deleteResponse.message } else { "success=false" }
                                                    Write-HostWithCapture "    ❌ DELETE FAILED - $errorMsg" -ForegroundColor Red
                                                }
                                            } catch {
                                                # Handle HTTP errors with detailed response
                                                $errorMessage = $_.Exception.Message
                                                
                                                if ($_.Exception.Response) {
                                                    try {
                                                        $responseStream = $_.Exception.Response.GetResponseStream()
                                                        $reader = New-Object System.IO.StreamReader($responseStream)
                                                        $responseBody = $reader.ReadToEnd()
                                                        $reader.Close()
                                                        $responseStream.Close()
                                                        
                                                        $errorJson = $responseBody | ConvertFrom-Json -ErrorAction SilentlyContinue
                                                        if ($errorJson -and $errorJson.message) {
                                                            $errorMessage += " | API Error: $($errorJson.message)"
                                                        } elseif ($errorJson -and $errorJson.error) {
                                                            $errorMessage += " | API Error: $($errorJson.error)"
                                                        } elseif ($responseBody -and $responseBody.Length -gt 0) {
                                                            $truncatedBody = if ($responseBody.Length -gt 200) { $responseBody.Substring(0, 200) + "..." } else { $responseBody }
                                                            $errorMessage += " | Response: $truncatedBody"
                                                        }
                                                    } catch {
                                                        # Ignore errors in error response parsing
                                                    }
                                                }
                                                
                                                Write-HostWithCapture "    ❌ DELETE ERROR - $errorMessage" -ForegroundColor Red
                                            }
                                        } catch {
                                            Write-HostWithCapture "    ❌ DELETE ERROR - Unexpected error: $($_.Exception.Message)" -ForegroundColor Red
                                        }
                                    } else {
                                        Write-HostWithCapture "    ❌ CREATE FAILED - success=false" -ForegroundColor Red
                                    }
                                } catch {
                                    Write-HostWithCapture "    ❌ CREATE ERROR - $($_.Exception.Message)" -ForegroundColor Red
                                }
                            }
                        } else {
                            Write-HostWithCapture "    ❌ READ FAILED - success=false" -ForegroundColor Red
                        }
                    } catch {
                        Write-HostWithCapture "    ❌ READ ERROR - $($_.Exception.Message)" -ForegroundColor Red
                    }
                }
            }
        } else {
            Write-HostWithCapture "  ❌ FAILED - success=false" -ForegroundColor Red
        }
    } catch {
        Write-HostWithCapture "  ❌ ERROR - $($_.Exception.Message)" -ForegroundColor Red
    }
}
}

# Product Domain Endpoints
$productEndpoints = @(
    "/api/product/product/list",
    "/api/product/collection/list",
    "/api/product/collection-plan/list",
    "/api/product/price-product/list",
    "/api/product/product-attribute/list",
    "/api/product/product-collection/list",
    "/api/product/product-plan/list",
    "/api/product/resource/list"
)

$filteredProductEndpoints = Filter-Endpoints -Endpoints $productEndpoints -Prefix $Prefix

if ($filteredProductEndpoints.Count -gt 0) {
    Write-HostWithCapture "`nPRODUCT DOMAIN" -ForegroundColor Yellow

foreach ($endpoint in $filteredProductEndpoints) {
    Write-HostWithCapture "Testing: $endpoint" -ForegroundColor Cyan
    try {
        $response = Invoke-RestMethod -Uri "$baseUrl$endpoint" -Method POST -Headers $headers -Body $body
        if ($response.success -eq $true) {
            $dataCount = if ($response.data) { $response.data.Count } else { 0 }
            Write-HostWithCapture "  ✅ SUCCESS - $dataCount items returned" -ForegroundColor Green
            
            # Test read endpoint if we have data
            if ($dataCount -gt 0 -and $response.data) {
                $firstItem = $response.data[0]
                if ($firstItem.id) {
                    $readEndpoint = $endpoint -replace "/list", "/read"
                    Write-HostWithCapture "    Testing read endpoint: $readEndpoint" -ForegroundColor DarkCyan
                    try {
                        $readBody = @{ data = @{ id = $firstItem.id } } | ConvertTo-Json -Depth 3
                        $readResponse = Invoke-RestMethod -Uri "$baseUrl$readEndpoint" -Method POST -Headers $headers -Body $readBody
                        if ($readResponse.success -eq $true) {
                            $readDataCount = if ($readResponse.data) { $readResponse.data.Count } else { 0 }
                            Write-HostWithCapture "    ✅ READ SUCCESS - $readDataCount items returned" -ForegroundColor DarkGreen
                            
                            # Test create endpoint using read data with new ID
                            if ($readResponse.data -and $readResponse.data.Count -gt 0) {
                                $createEndpoint = $endpoint -replace "/list", "/create"
                                Write-HostWithCapture "    Testing create endpoint: $createEndpoint" -ForegroundColor DarkMagenta
                                try {
                                    $createData = $readResponse.data[0] | ConvertTo-Json -Depth 10 | ConvertFrom-Json
                                    # Generate new ID for create operation
                                    $createData.id = [System.Guid]::NewGuid().ToString()
                                    if ($createData.name) { $createData.name += "_test_copy" }
                                    if ($createData.title) { $createData.title += "_test_copy" }
                                    
                                    $createBody = @{ data = $createData } | ConvertTo-Json -Depth 10
                                    $createResponse = Invoke-RestMethod -Uri "$baseUrl$createEndpoint" -Method POST -Headers $headers -Body $createBody
                                    if ($createResponse.success -eq $true) {
                                        Write-HostWithCapture "    ✅ CREATE SUCCESS - New item created" -ForegroundColor DarkGreen
                                        
                                        # Test delete endpoint using the created item
                                        $deleteEndpoint = $endpoint -replace "/list", "/delete"
                                        Write-HostWithCapture "    Testing delete endpoint: $deleteEndpoint" -ForegroundColor DarkRed
                                        try {
                                            $deleteBody = @{ data = @{ id = $createResponse.data[0].id } } | ConvertTo-Json -Depth 3
                                            
                                            # Use Invoke-WebRequest for better error handling
                                            try {
                                                $webResponse = Invoke-WebRequest -Uri "$baseUrl$deleteEndpoint" -Method POST -Headers $headers -Body $deleteBody -ContentType "application/json"
                                                $deleteResponse = $webResponse.Content | ConvertFrom-Json
                                                
                                                if ($deleteResponse.success -eq $true) {
                                                    Write-HostWithCapture "    ✅ DELETE SUCCESS - Item deleted" -ForegroundColor DarkGreen
                                                } else {
                                                    $errorMsg = if ($deleteResponse.message) { $deleteResponse.message } else { "success=false" }
                                                    Write-HostWithCapture "    ❌ DELETE FAILED - $errorMsg" -ForegroundColor Red
                                                }
                                            } catch {
                                                # Handle HTTP errors with detailed response
                                                $errorMessage = $_.Exception.Message
                                                
                                                if ($_.Exception.Response) {
                                                    try {
                                                        $responseStream = $_.Exception.Response.GetResponseStream()
                                                        $reader = New-Object System.IO.StreamReader($responseStream)
                                                        $responseBody = $reader.ReadToEnd()
                                                        $reader.Close()
                                                        $responseStream.Close()
                                                        
                                                        $errorJson = $responseBody | ConvertFrom-Json -ErrorAction SilentlyContinue
                                                        if ($errorJson -and $errorJson.message) {
                                                            $errorMessage += " | API Error: $($errorJson.message)"
                                                        } elseif ($errorJson -and $errorJson.error) {
                                                            $errorMessage += " | API Error: $($errorJson.error)"
                                                        } elseif ($responseBody -and $responseBody.Length -gt 0) {
                                                            $truncatedBody = if ($responseBody.Length -gt 200) { $responseBody.Substring(0, 200) + "..." } else { $responseBody }
                                                            $errorMessage += " | Response: $truncatedBody"
                                                        }
                                                    } catch {
                                                        # Ignore errors in error response parsing
                                                    }
                                                }
                                                
                                                Write-HostWithCapture "    ❌ DELETE ERROR - $errorMessage" -ForegroundColor Red
                                            }
                                        } catch {
                                            Write-HostWithCapture "    ❌ DELETE ERROR - Unexpected error: $($_.Exception.Message)" -ForegroundColor Red
                                        }
                                    } else {
                                        Write-HostWithCapture "    ❌ CREATE FAILED - success=false" -ForegroundColor Red
                                    }
                                } catch {
                                    Write-HostWithCapture "    ❌ CREATE ERROR - $($_.Exception.Message)" -ForegroundColor Red
                                }
                            }
                        } else {
                            Write-HostWithCapture "    ❌ READ FAILED - success=false" -ForegroundColor Red
                        }
                    } catch {
                        Write-HostWithCapture "    ❌ READ ERROR - $($_.Exception.Message)" -ForegroundColor Red
                    }
                }
            }
        } else {
            Write-HostWithCapture "  ❌ FAILED - success=false" -ForegroundColor Red
        }
    } catch {
        Write-HostWithCapture "  ❌ ERROR - $($_.Exception.Message)" -ForegroundColor Red
    }
}
}


# Subscription Domain Endpoints
$subscriptionEndpoints = @(
    "/api/subscription/subscription/list",
    "/api/subscription/balance/list",
    "/api/subscription/invoice/list",
    "/api/subscription/plan/list",
    "/api/subscription/plan-settings/list",
    "/api/subscription/price-plan/list"
)

$filteredSubscriptionEndpoints = Filter-Endpoints -Endpoints $subscriptionEndpoints -Prefix $Prefix

if ($filteredSubscriptionEndpoints.Count -gt 0) {
    Write-HostWithCapture "`nSUBSCRIPTION DOMAIN" -ForegroundColor Yellow

foreach ($endpoint in $filteredSubscriptionEndpoints) {
    Write-HostWithCapture "Testing: $endpoint" -ForegroundColor Cyan
    try {
        $response = Invoke-RestMethod -Uri "$baseUrl$endpoint" -Method POST -Headers $headers -Body $body
        if ($response.success -eq $true) {
            $dataCount = if ($response.data) { $response.data.Count } else { 0 }
            Write-HostWithCapture "  ✅ SUCCESS - $dataCount items returned" -ForegroundColor Green
            
            # Test read endpoint if we have data
            if ($dataCount -gt 0 -and $response.data) {
                $firstItem = $response.data[0]
                if ($firstItem.id) {
                    $readEndpoint = $endpoint -replace "/list", "/read"
                    Write-HostWithCapture "    Testing read endpoint: $readEndpoint" -ForegroundColor DarkCyan
                    try {
                        $readBody = @{ data = @{ id = $firstItem.id } } | ConvertTo-Json -Depth 3
                        $readResponse = Invoke-RestMethod -Uri "$baseUrl$readEndpoint" -Method POST -Headers $headers -Body $readBody
                        if ($readResponse.success -eq $true) {
                            $readDataCount = if ($readResponse.data) { $readResponse.data.Count } else { 0 }
                            Write-HostWithCapture "    ✅ READ SUCCESS - $readDataCount items returned" -ForegroundColor DarkGreen
                            
                            # Test create endpoint using read data with new ID
                            if ($readResponse.data -and $readResponse.data.Count -gt 0) {
                                $createEndpoint = $endpoint -replace "/list", "/create"
                                Write-HostWithCapture "    Testing create endpoint: $createEndpoint" -ForegroundColor DarkMagenta
                                try {
                                    $createData = $readResponse.data[0] | ConvertTo-Json -Depth 10 | ConvertFrom-Json
                                    # Generate new ID for create operation
                                    $createData.id = [System.Guid]::NewGuid().ToString()
                                    if ($createData.name) { $createData.name += "_test_copy" }
                                    if ($createData.title) { $createData.title += "_test_copy" }
                                    
                                    $createBody = @{ data = $createData } | ConvertTo-Json -Depth 10
                                    $createResponse = Invoke-RestMethod -Uri "$baseUrl$createEndpoint" -Method POST -Headers $headers -Body $createBody
                                    if ($createResponse.success -eq $true) {
                                        Write-HostWithCapture "    ✅ CREATE SUCCESS - New item created" -ForegroundColor DarkGreen
                                        
                                        # Test delete endpoint using the created item
                                        $deleteEndpoint = $endpoint -replace "/list", "/delete"
                                        Write-HostWithCapture "    Testing delete endpoint: $deleteEndpoint" -ForegroundColor DarkRed
                                        try {
                                            $deleteBody = @{ data = @{ id = $createResponse.data[0].id } } | ConvertTo-Json -Depth 3
                                            
                                            # Use Invoke-WebRequest for better error handling
                                            try {
                                                $webResponse = Invoke-WebRequest -Uri "$baseUrl$deleteEndpoint" -Method POST -Headers $headers -Body $deleteBody -ContentType "application/json"
                                                $deleteResponse = $webResponse.Content | ConvertFrom-Json
                                                
                                                if ($deleteResponse.success -eq $true) {
                                                    Write-HostWithCapture "    ✅ DELETE SUCCESS - Item deleted" -ForegroundColor DarkGreen
                                                } else {
                                                    $errorMsg = if ($deleteResponse.message) { $deleteResponse.message } else { "success=false" }
                                                    Write-HostWithCapture "    ❌ DELETE FAILED - $errorMsg" -ForegroundColor Red
                                                }
                                            } catch {
                                                # Handle HTTP errors with detailed response
                                                $errorMessage = $_.Exception.Message
                                                
                                                if ($_.Exception.Response) {
                                                    try {
                                                        $responseStream = $_.Exception.Response.GetResponseStream()
                                                        $reader = New-Object System.IO.StreamReader($responseStream)
                                                        $responseBody = $reader.ReadToEnd()
                                                        $reader.Close()
                                                        $responseStream.Close()
                                                        
                                                        $errorJson = $responseBody | ConvertFrom-Json -ErrorAction SilentlyContinue
                                                        if ($errorJson -and $errorJson.message) {
                                                            $errorMessage += " | API Error: $($errorJson.message)"
                                                        } elseif ($errorJson -and $errorJson.error) {
                                                            $errorMessage += " | API Error: $($errorJson.error)"
                                                        } elseif ($responseBody -and $responseBody.Length -gt 0) {
                                                            $truncatedBody = if ($responseBody.Length -gt 200) { $responseBody.Substring(0, 200) + "..." } else { $responseBody }
                                                            $errorMessage += " | Response: $truncatedBody"
                                                        }
                                                    } catch {
                                                        # Ignore errors in error response parsing
                                                    }
                                                }
                                                
                                                Write-HostWithCapture "    ❌ DELETE ERROR - $errorMessage" -ForegroundColor Red
                                            }
                                        } catch {
                                            Write-HostWithCapture "    ❌ DELETE ERROR - Unexpected error: $($_.Exception.Message)" -ForegroundColor Red
                                        }
                                    } else {
                                        Write-HostWithCapture "    ❌ CREATE FAILED - success=false" -ForegroundColor Red
                                    }
                                } catch {
                                    Write-HostWithCapture "    ❌ CREATE ERROR - $($_.Exception.Message)" -ForegroundColor Red
                                }
                            }
                        } else {
                            Write-HostWithCapture "    ❌ READ FAILED - success=false" -ForegroundColor Red
                        }
                    } catch {
                        Write-HostWithCapture "    ❌ READ ERROR - $($_.Exception.Message)" -ForegroundColor Red
                    }
                }
            }
        } else {
            Write-HostWithCapture "  ❌ FAILED - success=false" -ForegroundColor Red
        }
    } catch {
        Write-HostWithCapture "  ❌ ERROR - $($_.Exception.Message)" -ForegroundColor Red
    }
}
}

Write-HostWithCapture "`nTesting completed!" -ForegroundColor Green

# Save results to file if ResultsFile parameter is provided
if ($ResultsFile) {
    # Create results directory if it doesn't exist
    $resultsDir = Split-Path $ResultsFile -Parent
    if (-not (Test-Path $resultsDir)) {
        New-Item -ItemType Directory -Path $resultsDir -Force | Out-Null
    }

    # Capture all console output and save to file for cross-shell compatibility
    $output = @"
Results will be saved to: $ResultsFile
Testing all /list endpoints in Espyna API (Fiber Framework)
=============================================

$($TestResults | Out-String)

Testing completed!
Results saved to: $ResultsFile
"@

    # Write the captured output to the results file
    $output | Out-File -FilePath $ResultsFile -Encoding UTF8 -Force

    Write-HostWithCapture "Results saved to: $ResultsFile" -ForegroundColor Cyan
}