// axios setup with test fallback
const loc = window.location.href.indexOf("http://localhost/evcc/assets/") === 0 ? {
  protocol: "http:",
  hostname: "localhost",
  port: "7070",
} : window.location;

axios.defaults.baseURL = loc.protocol + "//" + loc.hostname + (loc.port ? ":" + loc.port : "") + "/api";
axios.defaults.headers.post['Content-Type'] = 'application/json';

let editor;
//
// Mixins
//

let formatter = {
  data: function () {
    return {
      fmtLimit: 100,
      fmtDigits: 1,
    }
  },
  methods: {
    round: function(num, precision) {
      var base = 10 ** precision;
      return (Math.round(num * base) / base).toFixed(precision);
    },
    fmt: function (val) {
      if (val === undefined || val === null) {
        return 0;
      }
      val = Math.abs(val);
      return val >= this.fmtLimit ? this.round(val / 1e3, this.fmtDigits) : this.round(val, 0);
    },
    fmtUnit: function (val) {
      return Math.abs(val) >= this.fmtLimit ? "k" : "";
    },
    fmtDuration: function (d) {
      if (d <= 0 || d == null) {
        return '—';
      }
      var seconds = "0" + (d % 60);
      var minutes = "0" + (Math.floor(d / 60) % 60);
      var hours = "" + Math.floor(d / 3600);
      if (hours.length < 2) {
        hours = "0" + hours;
      }
      return hours + ":" + minutes.substr(-2) + ":" + seconds.substr(-2);
    },
    fmtShortDuration: function (d) {
      if (d <= 0 || d == null) {
        return '—';
      }
      var minutes = (Math.floor(d / 60) % 60);
      var hours = Math.floor(d / 3600);
      var tm;
      if (hours >= 1) {
        minutes = "0" + minutes;
        tm = hours + ":" + minutes.substr(-2);
      } else {
        var seconds = "0" + (d % 60);
        tm = minutes + ":" + seconds.substr(-2);
      }
      return tm;
    },
    fmtShortDurationUnit: function (d) {
      if (d <= 0 || d == null) {
        return '';
      }
      var hours = Math.floor(d / 3600);
      if (hours >= 1) {
        return "h";
      }
      return "m";
    },
  }
}

//
// State
//

let store = {
  state: {
    availableVersion: null,
    loadpoints: [],
  },
  update: function(msg) {
    let target = this.state;
    if (msg.loadpoint !== undefined) {
      while (this.state.loadpoints.length <= msg.loadpoint) {
        this.state.loadpoints.push({});
      }
      target = this.state.loadpoints[msg.loadpoint];
    }

    Object.keys(msg).forEach(function (k) {
      if (typeof toasts[k] === "function") {
        toasts[k]({message: msg[k]})
      } else {
        Vue.set(target, k, msg[k]);
      }
    });
  },
  init: function() {
    axios.get("config").then(function(msg) {
      for (let i=0; i<msg.data.loadpoints.length; i++) {
        let data = Object.assign(msg.data.loadpoints[i], { loadpoint: i });
        this.update(data);
      }

      delete msg.data.loadpoints;
      this.update(msg.data);
    }.bind(this)).catch(toasts.error);
  }
};

//
// Heartbeat
//

window.setInterval(function() {
  axios.get("health").catch(function(res) {
    res.message = "Server unavailable";
    toasts.error(res)
  });
}, 5000);

//
// Components
//

const toasts = new Vue({
  el: "#toasts",
  data: {
    items: {},
    count: 0,
  },
  methods: {
    raise: function (msg) {
      let found = false;
      Object.keys(this.items).forEach(function (k) {
        let m = this.items[k];
        if (m.type == msg.type && m.message == msg.message) {
          found = true;
        }
      }, this);
      if (!found) {
        msg.id = this.count++;
        Vue.set(this.items, msg.id, msg);
      }
    },
    error: function (msg) {
      msg.type = "error";
      this.raise(msg)
    },
    warn: function (msg) {
      msg.type = "warn";
      this.raise(msg);
    },
    remove: function (msg) {
      Vue.delete(this.items, msg.id);
    },
  }
});

Vue.component('message-toast', {
  template: '#message-template',
  props: ['item'],
  mounted: function () {
    const id = "#message-id-" + this.item.id;
    $(id).toast('show');
    $(id).on('hidden.bs.toast', function () {
      toasts.remove(this.item);
    }.bind(this))
  },
});

Vue.component('version', {
  template: '#version-template',
  props: ['installed'],
  data: function () {
    return {
      state: store.state,
      notesShown: false,
    };
  },
  mounted: function () {
    $(this.$refs.notes)
      .on('show.bs.collapse', function () { this.notesShown = true; }.bind(this))
      .on('hide.bs.collapse', function () { this.notesShown = false; }.bind(this));
  },
  watch: {
    "state.availableVersion": function () {
      if (this.installed != "<<.Version>>" && // go template parsed?
        this.installed != "0.0.1-alpha" && // make used?
        this.state.availableVersion != this.installed) {
        $(this.$refs.bar).collapse("show");
      }
    }
  }
});

Vue.component('setup-banner', {
  template: '#setup-banner-template',
  props: ['setup']
});

Vue.component('site', {
  template: '#site-template',
  props: ['state'],
  mixins: [formatter],
  computed: {
    multi: function() {
      return this.state.loadpoints.length > 1
    }
  },
  methods: {
    connect: function() {
      const protocol = loc.protocol == "https:" ? "wss:" : "ws:";
      const uri = protocol + "//" + loc.hostname + (loc.port ? ":" + loc.port : "") + "/ws";
      const ws = new WebSocket(uri), self = this;
      ws.onerror = function(evt) {
        ws.close();
      };
      ws.onclose = function(evt) {
        window.setTimeout(self.connect, 1000);
      };
      ws.onmessage = function(evt) {
        try {
          var msg = JSON.parse(evt.data);
          store.update(msg);
        }
        catch (e) {
          toasts.error(e, evt.data)
        }
      };
    }
  },
  created: function() {
    this.connect();
  }
});

Vue.component("site-details", {
  template: "#site-details-template",
  props: ["state"],
  mixins: [formatter]
});

Vue.component("loadpoint", {
  template: "#loadpoint-template",
  props: ["state", "id", "pv", "multi"],
  mixins: [formatter],
  data: function() {
    return {
      tickerHandle: null,
    };
  },
  computed: {
    hasTargetSoC: function () {
      return this.state.socLevels != null && this.state.socLevels.length > 0;
    },
  },
  watch: {
    "state.chargeDuration": function() {
      window.clearInterval(this.tickerHandle);
      // only ticker if actually charging
      if (this.state.charging && this.state.chargeDuration >= 0) {
        this.tickerHandle = window.setInterval(function() {
          this.state.chargeDuration += 1;
        }.bind(this), 1000);
      }
    },
  },
  methods: {
    api: function (func) {
      return "loadpoints/" + this.id + "/" + func;
    },
    targetMode: function (mode) {
      axios.post(this.api("mode") + "/" + mode).then(function (response) {
        this.state.mode = response.data.mode;
      }.bind(this)).catch(toasts.error);
    },
    targetSoC: function (soc) {
      axios.post(this.api("targetsoc") + "/" + soc).then(function (response) {
        this.state.targetSoC = response.data.targetSoC;
      }.bind(this)).catch(toasts.error);
    },
  },
  destroyed: function() {
    window.clearInterval(this.tickerHandle);
  }
});

Vue.component("loadpoint-details", {
  template: "#loadpoint-details-template",
  props: ["state"],
  mixins: [formatter]
});

Vue.component("vehicle", {
  template: "#vehicle-template",
  props: ["state"],
  computed: {
    socChargeDisplayWidth: function () {
      if (this.state.soc && this.state.socCharge >= 0) {
        return this.state.socCharge;
      }
      return 100;
    },
    socChargeDisplayValue: function () {
      // no soc or no soc value
      if (!this.state.soc || this.state.socCharge < 0) {
        let chargeStatus = "getrennt";
        if (this.state.charging) {
          chargeStatus = "laden";
        } else if (this.state.connected) {
          chargeStatus = "verbunden";
        }
        return chargeStatus;
      }

      // percent value if enough space
      let socCharge = this.state.socCharge;
      if (socCharge >= 10) {
        socCharge += "%";
      }
      return socCharge;
    }
  }
});

Vue.component("mode", {
  template: "#mode-template",
  props: ["mode", "pv", "caption"],
  methods: {
    targetMode: function (mode) {
      this.$emit("updated", mode)
    }
  },
});

Vue.component("soc", {
  template: "#soc-template",
  props: ["soc", "caption", "levels"],
  computed: {
    levelsOrDefault: function() {
      if (this.levels == null || this.levels.length == 0) {
        return []; // disabled, or use 30, 50, 80, 100
      }
      return this.levels;
    }
  },
  methods: {
    targetSoC: function (mode) {
      this.$emit("updated", mode)
    }
  },
});

//
// Setup
//

const setup = Vue.component("setup", {
  template: "#setup-template",
  data: function () {
    return {
      state: store.state,
      templates: [
        { templateClass: "meter", title: "Zähler", data: [] },
        { templateClass: "charger", title: "Ladegeräte", data: [] },
        { templateClass: "vehicle", title: "EVs", data: [] },
      ],
      wizardSteps: [
        {
          step: 1, // the step number as shown on the page
          title: "Netzzähler", // the title of the step as shown on the page
          templateClass: "meter", // the template class to use for allowing the user to select items from
          description: "Ein Netzzähler wird benötigt um den PV Überschuss zu erkennen und damit das Laden zu steuern.",
        },
        {
          step: 2,
          title: "PV",
          templateClass: "meter",
          description: "Ein PV Zähler oder direkt der PV Wechselrichter ermöglicht die Anzeige wieviel des PV Stroms erzeugt wird.",
        },
        {
          step: 3,
          title: "Hausbatterie",
          templateClass: "meter",
          description: "Daten eines vorhandenen Batterie-Wechselrichter ermöglichen dessen Ladestrom für das Laden des EV zu berücksichtigen.",
        },
        {
          step: 4,
          title: "Ladegerät",
          templateClass: "charger",
          description: "Das Ladegerät welches gesteuert werden soll.",
        },
        {
          step: 5,
          title: "E-Auto",
          templateClass: "vehicle",
          description: "Die Angabe des E-Autos ermöglicht die Anzeige des aktuellen Ladezustandes.",
        },
      ],
      activeWizardStep: 1,
      activeTemplateClass: "",
      selectedItem: -1,
      editorInstance: null,
      errorMessage: "",
      currentTestIDInProgress: 0,
      testInProgress: false,
      testFailed: false,
      testSuccessful: false,
      wizardStepInitInProgress: false,
    };
  },
  computed: {
    activeWizardStepHasTemplateClass: function () {
      return this.wizardSteps[this.activeWizardStep - 1].templateClass != '';
    },
    activeWizardStepAllowNext: function() {
      if(this.activeWizardStep >= this.wizardSteps.length - 1) {
        return false;
      }
      return (this.testSuccessful == false);     
    },
    activeWizardStepTemplatesItem: function () {
      return this.templateByTemplateClass(this.wizardSteps[this.activeWizardStep - 1].templateClass);
    },
    testButtonInactive: function () {
      return (this.editorInstance != null && this.editorInstance.getValue().length == 0) || this.testInProgress == true
    },
    isTestInProgress: function () {
      return this.testInProgress == true
    },
    testSuccessMessageActive: function () {
      return this.testSuccessful == true
    },
    testFailureMessageActive: function () {
      return this.testFailed == true && this.testInProgress == false
    }
  },
  watch: {
    "activeTemplateClass": function () {
      this.updateTemplates(this.activeTemplateClass);
    },
    "selectedItem": function () {
      var templateItem = this.templateByTemplateClass(this.activeTemplateClass)
      var templateText = "";
      if (this.selectedItem >= 0)
        templateText = templateItem.data[this.selectedItem].template;
      // in case the wizard step was already processed, we don't want to overwrite the user config with defaults
      if (this.wizardStepInitInProgress == true) {
        return
      }
      // Reset variables
      this.currentTestIDInProgress = 0;
      this.testInProgress = false;
      this.errorMessage = "";
      this.testSuccessful = false;
      this.testFailed = false;
      this.editorInstance.setValue(templateText);
    },
  },
  methods: {
    initWizardSteps: function () {
      // add internal variable defaults
      for (var i = 0; i < this.wizardSteps.length; i++) {
        var wizardStep = this.wizardSteps[i];
        wizardStep.selectedItem = -1; // stores the selected template index from the list
        wizardStep.configuration = ''; // stores the entered (and validated) configuration
      }
    },
    initEditor: function() {
      this.editorInstance = monaco.editor.create(document.getElementById('editorContainer'), {
        value: [
          ''
        ].join('\n'),
        minimap: { enabled: false },
        lineNumbers: "off",
        folding: false,
        language: 'yaml'
      });
      this.editorInstance.onDidChangeModelContent(event => {
        if (this.wizardStepInitInProgress == false) {
          this.testInProgress = false;
          this.errorMessage = "";
          this.testSuccessful = false;
          this.testFailed = false;
          this.testInProgress = false;
          this.testFailed = false;
        }
      });
    },
    resetEditorData: function () {
      this.currentTestIDInProgress = 0;
      this.testInProgress = false;
      this.testSuccessful = false;
      this.testFailed = false;
      this.errorMessage = "";
      this.editorInstance.setValue("");
    },
    selectWizardStep: function (newActiveStep) {
      this.wizardStepInitInProgress = true;
      this.resetEditorData();
      this.activeWizardStep = newActiveStep;
      var wizardStep = this.wizardSteps[this.activeWizardStep - 1];
      if (wizardStep.configuration != '') {
        this.testSuccessful = true;
        this.selectedItem = wizardStep.selectedItem;
      } else {
        this.selectedItem = -1;
      }
      this.editorInstance.setValue(wizardStep.configuration);
      this.activeTemplateClass = wizardStep.templateClass;      
    },
    templatesAPI: function (func) {
      return "config/templates/" + func;
    },
    validateAPI: function (func) {
      return "config/validate/" + func;
    },
    update: function (target, dataset) {
      target.data = [];
      while (target.data.length <= dataset) {
        target.data.push({});
      }
      for (let i = 0; i < dataset.length; i++) {
        var value = dataset[i];
        Vue.set(target.data, i, value);
      }
    },
    templateByTemplateClass: function (templateClass) {
      var templateItem;
      for (let i = 0; i < this.templates.length; i++) {
        if (this.templates[i].templateClass == templateClass) {
          templateItem = this.templates[i];
        }
      }
      return templateItem;
    },
    isActiveTemplateClass: function (templateClass) {
      return this.activeTemplateClass == templateClass;
    },
    updateTemplates: function (templateClass) {
      let templateItem = this.templateByTemplateClass(templateClass);
      if (templateItem !== undefined) {
        axios.get(this.templatesAPI(templateItem.templateClass)).then(function (msg) {
          this.update(templateItem, msg.data)
        }.bind(this)).catch(this.showErrorMessage);
      }
    },
    showErrorMessage: function (error) {
      this.errorMessage = error.message;
    },
    errorValidating: function (error) {
      this.testInProgress = false;
      this.testFailed = true;
      this.editorInstance.updateOptions({ readOnly: false });
      this.showErrorMessage(error);
    },
    cancelValidation: function () {
      this.testInProgress = false;
      this.testFailed = false;
      this.editorInstance.updateOptions({ readOnly: false });
    },
    checkValidation: function (validationID) {
      axios.get(this.validateAPI(validationID)).then(msg => {
        if (validationID != this.currentTestIDInProgress) {
          // ignore this, probably was a cancelled test
        } else if (!msg.data.completed) {
          window.setTimeout(this.checkValidation.bind(this, validationID), 1000);
        } else if (msg.data.completed) {
          // check if any data element has an error
          var errorMessage = "";
          if (msg.data.data) {
            for (const key in msg.data.data) {
              const element = msg.data.data[key];
              if (element.error) {
                if (errorMessage.length > 0) {
                  errorMessage = errorMessage + "; ";
                }
                errorMessage = errorMessage + key + ": " + element.error;
              }
            }
          } else if (msg.error) {
            errorMessage = msg.error;
          } else if (msg.data.error) {
            errorMessage = msg.data.error;
          } else {
            console.log(msg);
            errorMessage = "This should not happen, but we got an undefined state response from the server!";
          }
          if (errorMessage != "") {
            this.errorValidating({ message: errorMessage });
          } else {
            this.testInProgress = false;
            this.testSuccessful = true;
            // save the configuration
            var wizardStep = this.wizardSteps[this.activeWizardStep - 1];
            wizardStep.configuration = this.editorInstance.getValue();
            wizardStep.selectedItem = this.selectedItem;
            this.editorInstance.updateOptions({ readOnly: false });
          }
        } else {
          this.errorValidating({ message: "something went very wrong as this should not occur :(" });
        }
      }).catch(err => this.errorValidating(err));
    },
    validateConfig: function (event) {
      this.testSuccessful = false;
      this.testFailed = false;
      this.errorMessage = "";
      var templateText = this.editorInstance.getValue();
      if (templateText.length > 0) {
        var options = {
          headers: { 'content-type': 'text/plain'},
        }
        this.editorInstance.updateOptions({ readOnly: true });
        this.testInProgress = true;
        this.validationID = 0;
        axios.post(this.validateAPI(this.activeTemplateClass), templateText, options).then(msg => {
          var errorMessage = "";
          var validationID = 0;
          if (msg.data.id) {
            validationID = msg.data.id;
          } else {
            if (msg.data.error) {
              errorMessage = msg.data.error;
            } else {
              console.log(msg);
              errorMessage = "This should not happen, but we got an undefined state response from the server!";
            }
          }
          if (errorMessage != "") {
            this.errorValidating({ message: errorMessage });
          } else if (validationID > 0) {
            this.currentTestIDInProgress = validationID;
            window.setTimeout(this.checkValidation.bind(this, validationID), 1000);
          } else {
            this.testInProgress = false;
            this.editorInstance.updateOptions({ readOnly: false });
          }
        }).catch(err => this.errorValidating(err));
      }
    }
  },
  mounted: function () {
    this.initWizardSteps();
    this.initEditor();
    this.activeTemplateClass = this.templates[0].templateClass;
  },
  updated: function () {
    this.$nextTick(function () {
      // Code that will run only after the
      // entire view has been re-rendered
      this.wizardStepInitInProgress = false;
    });
  }
});

//
// Routing
//

const main = Vue.component('main', {
  template: '#main-template',
  data: function() {
    return {
      state: store.state  // global state
    }
  },
  methods: {
    configured: function (val) {
      // for development purposes
      if (val == '<<.Configured>>') {
        return true;
      }
      if (!isNaN(parseInt(val)) && parseInt(val) > 0) {
        return true;
      }
      return false;
    }
  }
});

const config = Vue.component("config", {
  template: "#config-template",
  data: function() {
    return {
      state: store.state // global state
    };
  },
});

const embed = Vue.component("embed", {
  template: "#embed-template",
  props: ["title", "subtitle", "img", "iframe", "link"],
});

const routes = [
  { path: "/", component: main },
].concat(routerLinks().map(function(props, idx) {
  return { path: "/links/" + idx, component: embed, props: props }
})).concat([
  { path: "/setup", component: setup },
  { path: "/config", component: config },
]);

const router = new VueRouter({
  routes, // short for `routes: routes`
  linkExactActiveClass: "active" // Bootstrap <nav>
});

const app = new Vue({
  router,
}).$mount("#app");

store.init();
