{{- define "indevolt-batterymode" }}
batterymode:
  source: switch
  switch:
    - case: 1 # normal
      set:
        source: sequence
        set:
          # Step 1: Set Grid Charging Setting to Disable (1143=0). Must precede the mode change: a write while it is pending makes the device drop it
          - source: http
            uri: "http://{{ .host }}:8080/rpc/Indevolt.SetData?config={\"f\":16,\"t\":1143,\"v\":[0]}"
            method: POST
            {{- include "auth" . | indent 12 }}
            jq: if type == "object" and .result == true then . else error("write rejected") end
          # Step 2: Set Mode to Self-consumed Prioritized (47005=1), which is the default for this battery
          - source: http
            uri: "http://{{ .host }}:8080/rpc/Indevolt.SetData?config={\"f\":16,\"t\":47005,\"v\":[1]}"
            method: POST
            {{- include "auth" . | indent 12 }}
            jq: if type == "object" and .result == true then . else error("write rejected") end
    - case: 2 # holddischarge power
      set:
        source: sequence
        set:
          # Step 1: Set Mode to Real-time control (47005=4), which is necessary to write State Setting (Step 2)
          - source: http
            uri: "http://{{ .host }}:8080/rpc/Indevolt.SetData?config={\"f\":16,\"t\":47005,\"v\":[4]}"
            method: POST
            {{- include "auth" . | indent 12 }}
            jq: if type == "object" and .result == true then . else error("write rejected") end
          # Step 2: Set State Setting to Standby (47015=0), additionally adjusting power and SOC to valid values to avoid API rejections
          # API validity: documented ranges are power 50 to 2400 W (47016)
          - source: http
            uri: "http://{{ .host }}:8080/rpc/Indevolt.SetData?config={\"f\":16,\"t\":47015,\"v\":[0,50,{{ .minsoc }}]}"
            method: POST
            {{- include "auth" . | indent 12 }}
            jq: if type == "object" and .result == true then . else error("write rejected") end
    - case: 3 # charge
      set:
        source: sequence
        set:
          # Step 1: Set Grid Charging Setting to Charging (1143=1). Must precede the mode change: a write while it is pending makes the device drop it
          - source: http
            uri: "http://{{ .host }}:8080/rpc/Indevolt.SetData?config={\"f\":16,\"t\":1143,\"v\":[1]}"
            method: POST
            {{- include "auth" . | indent 12 }}
            jq: if type == "object" and .result == true then . else error("write rejected") end
          # Step 2: Set Mode to Real-time control (47005=4), which is necessary to write State Setting (Step 3)
          - source: http
            uri: "http://{{ .host }}:8080/rpc/Indevolt.SetData?config={\"f\":16,\"t\":47005,\"v\":[4]}"
            method: POST
            {{- include "auth" . | indent 12 }}
            jq: if type == "object" and .result == true then . else error("write rejected") end
          # Step 3: Set State Setting to Charging (47015=1), additionally adjusting power and SOC to valid values, which is required by the API
          - source: http
            uri: "http://{{ .host }}:8080/rpc/Indevolt.SetData?config={\"f\":16,\"t\":47015,\"v\":[1,{{ .maxchargepower }},{{ .maxsoc }}]}"
            method: POST
            {{- include "auth" . | indent 12 }}
            jq: if type == "object" and .result == true then . else error("write rejected") end
{{- end }}
