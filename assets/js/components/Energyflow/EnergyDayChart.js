import Chart from 'chart.js/auto'

function createXLabels(type) {
    const labels = []
    if (type == 'minute') {
        for (let h = 0; h < 24; h++) {
            let hour = String(h).padStart(2, '0')
            for (let m = 0; m < 60; m++) {
                labels.push(hour + ":" + String(m).padStart(2, '0'))
            }
        }
    
    } else if (type == 'hour') {
        for (let h = 0; h < 24; h++) {
            let hour = String(h).padStart(2, '0')
            labels.push(hour + ":00")
        }
    }
    return labels;
}

const LABEL_MODE = 'minute';

export var currentChart = null;

const CHART_COLORS = {
    
    ToGrid:     'rgb(255,130,77)',    
    ToCars:     'rgb(0, 166, 255)',
    ToHeating:  'rgb(0,77,230)',    
    ToStorage:  'rgb(94,199,299)',
    ToHouse:    'rgb(153, 219, 255)',
    BatterySoC: 'rgb(45, 134, 45)',
    FromPvs:    'rgb(250,218,126)',   
    FromGrid:   'rgb(255,130,77)',    
    FromStorage:'rgb(140, 217, 140)',   
};
  
const CHART_ORDER = {
    
    FromGrid:     6,
    FromStorage:  7,
    FromPvs:      8,
    
    ToHouse:      1,
    ToStorage:    9,
    ToHeating:    2,
    ToCars:       3,
    ToGrid:       10,
    
    BatterySoC:   0,
};

const data = {
    labels: createXLabels(LABEL_MODE),
    datasets: [
        {
            order: CHART_ORDER.BatterySoC,
            label: 'Speicher KWh',
            data: [],
            borderColor: CHART_COLORS.BatterySoC,
            backgroundColor: CHART_COLORS.BatterySoC,
            fill: false,
            borderWidth: 2,
            yAxisID: 'yBattery'
        },
        {
            order: CHART_ORDER.ToHouse,
            label: 'Haus',
            data: [],
            borderColor: CHART_COLORS.ToHouse,
            backgroundColor: CHART_COLORS.ToHouse,
            fill: true,
            borderWidth: 1,
            yAxisID: 'yPower',
        },
        {
            order: CHART_ORDER.ToHeating,
            label: 'Heizung',
            data: [],
            borderColor: CHART_COLORS.ToHeating,
            backgroundColor: CHART_COLORS.ToHeating,
            fill: false,
            borderWidth: 1,
            yAxisID: 'yPower'
        },
        {
            order: CHART_ORDER.ToCars,
            label: 'Auto',
            data: [],
            borderColor: CHART_COLORS.ToCars,
            backgroundColor: CHART_COLORS.ToCars,
            fill: true,
            borderWidth: 1,
            yAxisID: 'yPower'
        },
        {
            order: CHART_ORDER.FromPvs,
            label: 'PV',
            data: [],
            borderColor: CHART_COLORS.FromPvs,
            backgroundColor: CHART_COLORS.FromPvs,
            fill: true,
            borderWidth: 1,
            yAxisID: 'yPower'
        },
        {
            order: CHART_ORDER.FromStorage,
            label: 'Speicher',
            data: [],
            borderColor: CHART_COLORS.FromStorage,
            backgroundColor: CHART_COLORS.FromStorage,
            fill: true,
            borderWidth: 1,
            yAxisID: 'yPower'
        },
        {
            order: CHART_ORDER.FromGrid,
            label: 'Netz',
            data: [],
            borderColor: CHART_COLORS.FromGrid,
            backgroundColor: CHART_COLORS.FromGrid,
            fill: true,
            borderWidth: 1,
            yAxisID: 'yPower'
        },
    ]
};

function getMaxY(data) {
    let maxY = 0;
    let maxLen = 0
    data.datasets.forEach(element => {
        maxLen = Math.max(maxLen, data.datasets.filter(element => element.yAxisID == 'yPower').length)
    });
    for (let i = 0; i < maxLen; i++) {
        let total = maxY
        total = data.datasets.filter(element => element.yAxisID == 'yPower').map(element => element.data[i]).reduce((sum, val) => sum + val, 0)
        maxY = Math.max(maxY, total);
    }
    return maxY;
}

const maxY = getMaxY(data);

const config = {
    type: 'line',
    data: data,
    options: {
        pointStyle: false,
        plugins: {
            title: {
              display: true,
              text: 'Energie Monitor'
            },
            tooltip: {
              mode: 'index'
            },
            legend: {
                position: 'top',
                labels: {
                    usePointStyle: true,
                    pointStyle: 'circle',
                    boxHeight: 5,
                }
            }
        },
        scales: {
            x: {
                min: 0,
                max: 24*60,
                display: false,
                title: {
                    display: true,
                    text: ''
                },
                ticks: {
                    source: 'labels',
                    display: false,
                    callback: function(value, index, ticks) {
                        let label = String(Math.trunc(value/60)).padStart(2, '0') + ":" + String(Math.trunc(value%60)).padStart(2, '0')
                        return index % 60 === 0 ? label : '';
                    },
                },
            },
            xHours: {
                display: true,
                min: 0,
                max: 24,
                ticks: {
                    callback: function(value, index, ticks) {
                        if (value === 24) {
                            value = 0
                        }
                        return String(Math.trunc(value)).padStart(2, '0') + ":00";
                    },
                }
            },
            yPower: {
                stacked: true,
                suggestedMax: maxY,
                title: {
                    display: false,
                    text: 'Erzeugung/Verbrauch (KW)'
                },
                ticks: {
                    callback: function(value, index, ticks) {
                        return (value / 1000) + " KW";
                    }
                }
            },
            yBattery: {
                stacked: false,
                suggestedMax: 10,
                suggestedMin: 0,
                title: {
                    color: CHART_COLORS.BatterySoC,
                    display: false,
                    text: 'Batterie (KWh)'
                },
                position: 'right',
                grid: {
                    color: CHART_COLORS.BatterySoC,
                    drawOnChartArea: false, // only want the grid lines for one axis to show up
                },
                ticks: {
                    color: CHART_COLORS.BatterySoC,
                    callback: function(value, index, ticks) {
                        return value + " KWh";
                    }
                },
                z: 50, // on top
            },
          }
      
    }
};

export function addData(values) {
    config.data.datasets.forEach(item => {
        let date = new Date(values[0])
        
        let index = date.getHours();
        if (LABEL_MODE == 'minute') {
            index = (date.getHours() * 60) + date.getMinutes();
        }
        // 0 date, 1 FromPvs, 2 FromStorage, 3 FromGrid, 4 ToGrid, 5 ToStorage, 6 ToHouse, 7 ToHeating, 8 ToCars, 9 BatterySoC
        //                ToGrid + ToStorage + ToHouse + ToHeating + ToCars
        let consumption = values[4]+values[5]+values[6]+values[7]+values[8];
        //               FromPvs + FromStorage + FromGrid
        let production = values[1]+values[2]+values[3];
        if (consumption != production) {
            // FromGrid = consumption - (FromPvs + FromStorage)
            values[3] = consumption - (values[1]+values[2]);
        }
        if (item.order == CHART_ORDER.FromPvs) {
            item.data[index] = (values[1]);
        } else if (item.order == CHART_ORDER.FromStorage) {
            if (values[2] > 0) {
                item.data[index] = (values[2]);            
            } else if (values[5] > 0) {
                item.data[index] = (-values[5]);            
            } else {
                item.data[index] = 0
            }
        } else if (item.order == CHART_ORDER.FromGrid) {
            if (values[3] > 0) {
                item.data[index] = (values[3]);            
            } else if (values[4] > 0) {
                item.data[index] = (-values[4]);            
            } else {
                item.data[index] = 0
            }
        } else if (item.order == CHART_ORDER.ToGrid) {
            //item.data[index] = (-values[4]);            
            item.data[index] = 0
        } else if (item.order == CHART_ORDER.ToStorage) {
            //item.data[index] = (-values[5]);
            item.data[index] = 0
        } else if (item.order == CHART_ORDER.ToHouse) {
            item.data[index] = (-values[6]);            
        } else if (item.order == CHART_ORDER.ToHeating) {
            item.data[index] = (-values[7]);            
        } else if (item.order == CHART_ORDER.ToCars) {
            item.data[index] = (-values[8]);            
        } else if (item.order == CHART_ORDER.BatterySoC) {
            item.data[index] = (values[9]/10);            
        }
    });
}

export function calculateDataAndOptions(data) {
    var d = new Date(data[0].timePoint)
    config.options.plugins.title.text = "Energie Monitor " + d.getDate()+"."+(d.getMonth()+1)+"."+d.getFullYear()
    // FromPvs, FromStorage, FromGrid, ToGrid, ToStorage, ToHouse, ToHeating, ToCars, BatterySoC
    data.forEach(item => {
        const dataToAdd = [
            item.timePoint,
            parseFloat(item.fromPvs),
            parseFloat(item.fromStorage),
            parseFloat(item.fromGrid),
            parseFloat(item.toGrid),
            parseFloat(item.toStorage),
            parseFloat(item.toHouse),
            parseFloat(item.toHeating),
            parseFloat(item.toCars),
            parseFloat(item.batterySoC),
        ];
        addData(dataToAdd)
    })
    let socIdx = -1;
    for (let n = 0; n<config.data.datasets.length; n++) {
        if (config.data.datasets[n].order == CHART_ORDER.BatterySoC) {
            socIdx = n;   
            break;        
        }
    }
    var d = new Date(data[data.length-1].timePoint)
    var dataLength = (d.getHours() * 60) + d.getMinutes()
    for (let i = 0; i<dataLength; i++) {
        if ((config.data.datasets[0].data[i] == null) && (i > 0)) {
            config.data.datasets.forEach(item => {
                item.data[i] = item.data[i-1];
            });    
        } else if ((config.data.datasets[socIdx].data[i] != null) && (i > 0) && (socIdx >= 0) && (config.data.datasets[socIdx].data[i] <= 0)) {
            if (Math.abs(config.data.datasets[socIdx].data[i]-config.data.datasets[socIdx].data[i-1]) > 1) {
                config.data.datasets[socIdx].data[i] = config.data.datasets[socIdx].data[i-1];
            }
        }
    }
    var fromStorageSum = 0;
    var toStorageSum = 0;

    var fromGridSum = 0;
    var toGridSum = 0;

    config.data.datasets.forEach(item => {
        if (item.order == CHART_ORDER.FromPvs) {
            var sum = 0;
            item.data.forEach(val => {
                if (val) {
                    sum += val;
                }
            })
            item.label = "PV (+" + parseInt(sum/60/1000) + " KWh)";
        } else if (item.order == CHART_ORDER.FromStorage) {
            var to = 0;
            var from = 0;
            item.data.forEach(val => {
                if (val) {
                    if (val < 0) {
                        to += val;
                    } else {
                        from += val;
                    }
                }
            })
            item.label = "Speicher (" + parseInt(to/60/1000) + "/+" + parseInt(from/60/1000) + " KWh)";
        } else if (item.order == CHART_ORDER.FromGrid) {
            var to = 0;
            var from = 0;
            item.data.forEach(val => {
                if (val) {
                    if (val < 0) {
                        to += val;
                    } else {
                        from += val;
                    }
                }
            })
            item.label = "Netz (" + parseInt(to/60/1000) + "/+" + parseInt(from/60/1000) + " KWh)";
        } else if (item.order == CHART_ORDER.ToHouse) {
            var sum = 0;
            item.data.forEach(val => {
                if (val) {
                    sum += val;
                }
            })
            item.label = "Haus (" + parseInt(sum/60/1000) + " KWh)";
        } else if (item.order == CHART_ORDER.ToHeating) {
            var sum = 0;
            item.data.forEach(val => {
                if (val) {
                    sum += val;
                }
            })
            item.label = "Heizung (" + parseInt(sum/60/1000) + " KWh)";
        } else if (item.order == CHART_ORDER.ToCars) {
            var sum = 0;
            item.data.forEach(val => {
                if (val) {
                    sum += val;
                }
            })
            item.label = "Auto (" + parseInt(sum/60/1000) + " KWh)";
        }
    });
    let maxY = getMaxY(config.data);
    config.options.scales.yPower.suggestedMax = maxY
}

export async function getConfig(data=null, json=null, host="192.168.1.32", port="7070") {
    let resultPromise = null

    if (data == null && json == null) {
        const today = new Date(Date.now());
        const offset = Math.abs(today.getTimezoneOffset()/60);
        const url = "http://"+host+":"+port+"/api/data/"+today.getFullYear()+"/"+(today.getMonth()+1)+"/"+today.getDate()+"/"+offset;
        resultPromise = fetch(url)
            .then(response => response.json())
            .then(data => {
                calculateDataAndOptions(data.result)
            })
            .then(data => {
                return { data: config.data, options: config.options };
            });
    } else if (json != null) {
        resultPromise = new Promise(resolve => {
            calculateDataAndOptions(JSON.parse(json).result),
            resolve({ data: config.data, options: config.options });
        })
    } else {
        resultPromise = new Promise(resolve => {
            calculateDataAndOptions(data),
            resolve({ data: config.data, options: config.options });
        })
    }
    return resultPromise;
}