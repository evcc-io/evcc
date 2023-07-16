# Update/-grade node

[Update node.js](https://stackoverflow.com/questions/8191459/how-do-i-update-node-js)

```
brew update
brew upgrade
```

```
brew install nvm
```

```
source $(brew --prefix nvm)/nvm.sh
```

```
nvm install <node version>
```

```
nvm list available
nvm use <node version>
```

Aktuelle Version unter https://nodejs.org

```
npm install xxxx
```

# Wenn `make ui` failed ...

Meistens alte Import-Abhängigkeiten, mit `npm install xxx lösbar`

# Start evcc

## SQLite nicht mehr als Parameter oder in Config YAML

```
EVCC_DATABASE_DSN=./evcc.db ./evcc --config evcc.yaml
```

# SQLite 

[Download Übersicht](https://sqlitebrowser.org/dl/)

[Download DMG](https://download.sqlitebrowser.org/DB.Browser.for.SQLite-arm64-3.12.2.dmg)

````
brew install --cask db-browser-for-sqlite
````

```
sudo apt install sqlite3
```


`brew` installiert eine `DB Browser for SQLite.app` -> starten -> `evcc.db` öffnen

[Doku](https://github.com/sqlitebrowser/sqlitebrowser/wiki)

[CLI](https://sqlite.org/cli.html)

# GO ORM 

https://gorm.io/docs/connecting_to_the_database.html
https://gorm.io/docs/query.html

# GO Language Reference

https://go.dev/ref/spec

['main' Funktion](https://stackoverflow.com/questions/42333488/package-main-and-func-main)

# Debug GO in VCode

https://github.com/golang/vscode-go/wiki/debugging

Statt `"${fileDirname}"` auch `main.go`

```
{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "evcc",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${fileDirname}"
        }
    ]
}
```
# Chart.js

https://github.com/chartjs/Chart.js

https://www.chartjs.org/docs/latest/samples/information.html

"Alternatively, you can run them locally. To do so, clone the Chart.js repository from GitHub, run `pnpm ci` to install all packages, then run 'pnpm run docs:dev' to build the documentation. As soon as the build is done, you can go to http://localhost:8080/samples to see the samples."

https://www.chartjs.org/docs/latest/configuration/

https://www.chartjs.org/docs/latest/samples/line/multi-axis.html

https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide

https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/padStart

https://rgbcolorcode.com/color/004D0D

1 timestamp,
2 produced_energy,
3 consumed_energy,
4 battery_charged_energy,
5 battery_discharged_energy,
6 grid_feedin_energy,
7 grid_purchase_energy
8 (aus 'measurements.csv')

   1 -> 1 Timestamp;
   2 -> 2 FromPvs;
   5 -> 3 FromStorage;
   7 -> 4 FromGrid;
   6 -> 5 ToGrid;
   4 -> 6 ToStorage;
   3 -> 7 ToHouse;
   0 -> 8 ToHeating;
   0 -> 9 ToCars;
   8 -> 10 BatterySoC

grep "^2023-03-10T" statistics.csv | tr ',' ' ' | while read line; do echo "${line} $(grep "$(echo "${line}" | cut -d' ' -f1)" measurements.csv | cut -d',' -f8)"; done |  awk '{printf("addData([\"%s\", %s, %s, %s, %s, %s, %s, %s, %s, %s]);\n", $1, $2, $5, $7, $6, $4, $3, 0, 0, $8)}' 

## SQLLite Export in data umwandeln
grep -v "^id" evcc.csv | tr ',' ' ' | cut -d ' ' -f2-99 | awk '{printf("addData([\"%s-%02d-%02dT%02d:%02d:00+01:00\",%d,%d,%d,%d,%d,%d,%d,%d,%d])\n",$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)}'

# Repo & Branch

https://github.com/evcc-io/evcc/

Branch: 
* master
* feature/ETREL_INCH_HOME-patch

# TODO

* Zeiten:
  * DONE: Zeit in DB = UTC? -> wenn nicht, sollte UTC sein.
  * DONE: Zeit aus DB = UTC? -> wenn nicht, sollte UTC sein
  * DONE: Zeit Anzeige = local time? -> sollte als UTC kommen und auf local time umgewandelt werden.
  * Local Time 2023-04-27 00:00 bis 23:59 ist UTC früher! D.h. 