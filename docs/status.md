# Charger status transition events

From-To|A|B|C
-|-|-|-
None<br>(Startup)|disconnected|connected|connected<br>start charging
A|-|connected|connected<br>start charging
B|disconnected|-|start charging
C|disconnected<br>stop charging|stop charging|-
