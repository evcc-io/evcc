<template>
  <Line v-if="displayChart" :data="chartData" :options="chartOptions" />
</template>

<script>
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend
} from 'chart.js'
import { Line } from 'vue-chartjs'
import { getConfig as getConfigFromToday } from './EnergyDayChart.js'
import { isProxy, toRaw } from 'vue';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend
)

export default {
  name: 'EnergyDayChart',
  components: {
    Line
  },
  props: {
    powerData: { type: String },
  },
  data: () => ({
    chartDataLoaded: false,
    chartData: null,
    chartOptions: null,
    resultData: null,
    lastFetch: null,
  }),
  updated() {
    this.loadAndCalculateData();
  },
  async mounted() {
    this.loadAndCalculateData();
  },
  computed: {
    displayChart() {
      return this.chartDataLoaded && this.$parent.chartVisible;
    }
  },
  methods: {
    loadAndCalculateData() {
      if (this.lastFetch && (Date.now()-this.lastFetch) < 10000) {
        return;
      }
      try {
        var jsonData = "{ \"result\": "+this.powerData+"}"
        if (!this.powerData || this.powerData.length == 0) {
          jsonData = "{ \"result\": []}"
        }
        getConfigFromToday(null, jsonData)
          .then(resultData => {
            this.resultData = resultData;
            this.lastFetch = Date.now();
            this.chartData = resultData.data;
            this.chartOptions = resultData.options;
            if (this.chartDataLoaded == false) {
              this.chartDataLoaded = true;
            }
          });
      } catch(exception) {
        alert(exception);
      }
    }
  }
}
</script>
