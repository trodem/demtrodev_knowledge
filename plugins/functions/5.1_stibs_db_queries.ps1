# Config base MariaDB container
$env:DM_DB_CONTAINER = "mariadb"
$env:DM_DB_USER = "stibo"
$env:DM_DB_PASSWORD = "stibs"
$env:DM_DB_NAME = "stibs"

<#
.SYNOPSIS
Verifica se il container MariaDB è attivo.

.DESCRIPTION
Controlla che il container Docker configurato sia in esecuzione.

.EXAMPLE
dm db_status
#>
function db_status {
    _assert_command_available -Name docker
    docker ps --filter "name=$env:DM_DB_CONTAINER"
}


<#
.SYNOPSIS
Esegue una query MariaDB nel container Docker.

.DESCRIPTION
Usa docker exec per eseguire il client mysql e lanciare la query.

.PARAMETER Query
Query SQL da eseguire.

.EXAMPLE
dm db_query "SELECT * FROM users"
#>
function db_query {
    param(
        [Parameter(Mandatory)]
        [string]$Query
    )

    docker exec -i $env:DM_DB_CONTAINER `
        mysql -u$env:DM_DB_USER -p$env:DM_DB_PASSWORD `
        $env:DM_DB_NAME -e $Query
}


<#
.SYNOPSIS
Mostra database MariaDB.

.DESCRIPTION
Elenca tutti i database disponibili nel container.

.EXAMPLE
dm db_databases
#>
function db_databases {
    docker exec -i $env:DM_DB_CONTAINER `
        mysql -u$env:DM_DB_USER -p$env:DM_DB_PASSWORD `
        -e "SHOW DATABASES;"
}


<#
.SYNOPSIS
Mostra schema tabella.

.DESCRIPTION
Visualizza la struttura (columns, types) di una tabella.

.PARAMETER Table
Nome tabella.

.EXAMPLE
dm db_schema users
#>
function db_schema {
    param(
        [Parameter(Mandatory)]
        [string]$Table
    )

    db_query "DESCRIBE $Table;"
}


<#
.SYNOPSIS
Conta righe di una tabella.

.DESCRIPTION
Restituisce il numero totale di record presenti.

.PARAMETER Table
Nome tabella.

.EXAMPLE
dm db_count users
#>
function db_count {
    param(
        [Parameter(Mandatory)]
        [string]$Table
    )

    db_query "SELECT COUNT(*) AS total FROM $Table;"
}



<#
.SYNOPSIS
Apre shell MariaDB nel container.

.DESCRIPTION
Avvia sessione mysql interattiva dentro il container Docker.

.EXAMPLE
dm db_shell
#>
function db_shell {
    docker exec -it $env:DM_DB_CONTAINER `
        mysql -u$env:DM_DB_USER -p$env:DM_DB_PASSWORD `
        $env:DM_DB_NAME
}

<#
.SYNOPSIS
Dump database MariaDB.

.DESCRIPTION
Esporta l'intero database in un file .sql.

.PARAMETER Output
File di output.

.EXAMPLE
dm db_dump backup.sql
#>
function db_dump {
    param(
        [Parameter(Mandatory)]
        [string]$Output
    )

    docker exec $env:DM_DB_CONTAINER `
        mysqldump -u$env:DM_DB_USER -p$env:DM_DB_PASSWORD `
        $env:DM_DB_NAME > $Output
}



<#
.SYNOPSIS
Mostra tabelle del database.

.DESCRIPTION
Elenca tutte le tabelle del database configurato.

.EXAMPLE
dm db_tables
#>
function db_tables {
    db_query "SHOW TABLES;"
}


<#
.SYNOPSIS
Importa dump SQL.

.DESCRIPTION
Carica un file .sql nel database MariaDB.

.PARAMETER File
File SQL da importare.

.EXAMPLE
dm db_import dump.sql
#>
function db_import {
    param(
        [Parameter(Mandatory)]
        [string]$File
    )

    Get-Content $File | docker exec -i $env:DM_DB_CONTAINER `
        mysql -u$env:DM_DB_USER -p$env:DM_DB_PASSWORD `
        $env:DM_DB_NAME
}



<#
.SYNOPSIS
Esegue query MySQL dentro container Docker.

.DESCRIPTION
Utilizza docker exec per lanciare il client mysql e eseguire una query.

.PARAMETER Container
Nome container MySQL.

.PARAMETER User
Utente MySQL.

.PARAMETER Password
Password MySQL.

.PARAMETER Database
Database target.

.PARAMETER Query
Query SQL da eseguire.

.EXAMPLE
dm mysql_query mysql-db root password mydb "SELECT * FROM users"
#>
function mysql_query {
    param(
        [Parameter(Mandatory)][string]$Container,
        [Parameter(Mandatory)][string]$User,
        [Parameter(Mandatory)][string]$Password,
        [Parameter(Mandatory)][string]$Database,
        [Parameter(Mandatory)][string]$Query
    )

    docker exec -i $Container mysql -u$User -p$Password $Database -e $Query
}



<#
.SYNOPSIS
Elenca tabelle MySQL.

.DESCRIPTION
Mostra le tabelle presenti in un database MySQL nel container.

.PARAMETER Container
Container MySQL.

.PARAMETER User
Utente.

.PARAMETER Password
Password.

.PARAMETER Database
Database target.

.EXAMPLE
dm mysql_tables mysql-db root password mydb
#>
function mysql_tables {
    param(
        [Parameter(Mandatory)][string]$Container,
        [Parameter(Mandatory)][string]$User,
        [Parameter(Mandatory)][string]$Password,
        [Parameter(Mandatory)][string]$Database
    )

    mysql_query $Container $User $Password $Database "SHOW TABLES;"
}




<#
.SYNOPSIS
Dump database MySQL.

.DESCRIPTION
Esegue export completo database dal container.

.PARAMETER Container
Container MySQL.

.PARAMETER User
Utente.

.PARAMETER Password
Password.

.PARAMETER Database
Database.

.PARAMETER Output
File di output.

.EXAMPLE
dm mysql_dump mysql-db root password mydb dump.sql
#>
function mysql_dump {
    param(
        [Parameter(Mandatory)][string]$Container,
        [Parameter(Mandatory)][string]$User,
        [Parameter(Mandatory)][string]$Password,
        [Parameter(Mandatory)][string]$Database,
        [Parameter(Mandatory)][string]$Output
    )

    docker exec $Container mysqldump -u$User -p$Password $Database > $Output
}


<#
.SYNOPSIS
Individua container MariaDB da docker-compose.

.DESCRIPTION
Recupera l'ID del container associato al servizio mariadb.

.EXAMPLE
dm db.container
#>
function db_container {
    _assert_command_available -Name docker
    docker compose ps -q mariadb
}


<#
.SYNOPSIS
Recupera variabili ambiente MariaDB.

.DESCRIPTION
Legge MYSQL_USER, MYSQL_PASSWORD e MYSQL_DATABASE dal container.

.EXAMPLE
dm db.env
#>
function db_env {
    $container = db_container
    docker inspect $container |
        ConvertFrom-Json |
        Select-Object -ExpandProperty Config |
        Select-Object -ExpandProperty Env
}



<#
.SYNOPSIS
Apre HeidiSQL già connesso al MariaDB in docker-compose.

.DESCRIPTION
Legge automaticamente:
- container mariadb
- variabili environment
- porta esposta

e avvia HeidiSQL con connessione pronta.

.EXAMPLE
dm db_heidi
#>
function db_heidi {

    _assert_command_available -Name docker

    # Perce61 1. trova container
    $container = docker compose ps -q mariadb

    if (-not $container) {
        Write-Host ">> Container mariadb non trovato"
        return
    }

    # 2. leggi env dal container
    $envs = docker inspect $container |
        ConvertFrom-Json |
        Select-Object -ExpandProperty Config |
        Select-Object -ExpandProperty Env

    $user = ($envs | Where-Object {$_ -like "MYSQL_USER=*"}).Split("=")[1]
    $pass = ($envs | Where-Object {$_ -like "MYSQL_PASSWORD=*"}).Split("=")[1]
    $db   = ($envs | Where-Object {$_ -like "MYSQL_DATABASE=*"}).Split("=")[1]

    # fallback root
    if (-not $user) {
        $user = "root"
        $pass = ($envs | Where-Object {$_ -like "MYSQL_ROOT_PASSWORD=*"}).Split("=")[1]
    }

    # 3. porta esposta
    $port = docker inspect $container |
        ConvertFrom-Json |
        Select-Object -ExpandProperty NetworkSettings |
        Select-Object -ExpandProperty Ports |
        Select-Object -ExpandProperty "3306/tcp" |
        Select-Object -ExpandProperty HostPort

    if (-not $port) { $port = 3306 }

    # 4. percorso HeidiSQL
    $heidi = "C:\Program Files\HeidiSQL\heidisql.exe"
    _assert_path_exists -Path $heidi

    # 5. lancia HeidiSQL con connessione
    Start-Process $heidi `
        "-h=127.0.0.1 -u=$user -p=$pass -P=$port -d=$db"
}
