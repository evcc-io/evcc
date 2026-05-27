| fetchedSoc | chargedEnergy | result                                              |
| ---------- | ------------- | --------------------------------------------------- |
| nil        | <=0           | 0                                                   |
| nil        | value         | prevsoc + delta                                     |
| value      | <=0           | initialsoc setzen                                   |
| value      | value         | initialsoc/initialenergy setzen falls nicht gesetzt |
