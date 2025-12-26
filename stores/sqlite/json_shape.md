# JSON shape (kind discriminators)

## ReportX export/import

```json
{
  "id": 101,
  "reportFileId": 77,
  "game": "TN3.1",
  "clanNo": "0346",
  "turnNo": 90306,
  "createdAt": "2025-12-21T12:00:00Z",
  "units": [
    {
      "unitId": "0346",
      "turnNo": 90306,
      "startTN": "QQ 0203",
      "endTN": "QQ 0303",
      "src": {"docId":77,"unitId":"0346","turnNo":90306},
      "acts": [
        {
          "seq": 1,
          "kind": "follow",
          "targetUnitId": "0123",
          "src": {"docId":77,"unitId":"0346","turnNo":90306,"actSeq":1}
        },
        {
          "seq": 2,
          "kind": "goto",
          "destTN": "QQ 1010",
          "src": {"docId":77,"unitId":"0346","turnNo":90306,"actSeq":2}
        },
        {
          "seq": 3,
          "kind": "move",
          "steps": [
            {"seq": 1, "kind":"adv", "dir":"NE", "ok":true,
             "src":{"docId":77,"unitId":"0346","turnNo":90306,"actSeq":3,"stepSeq":1}},
            {"seq": 2, "kind":"adv", "dir":"E", "ok":false, "failWhy":"terrain",
             "src":{"docId":77,"unitId":"0346","turnNo":90306,"actSeq":3,"stepSeq":2}},
            {"seq": 3, "kind":"still",
             "src":{"docId":77,"unitId":"0346","turnNo":90306,"actSeq":3,"stepSeq":3}}
          ]
        },
        {
          "seq": 4,
          "kind": "scout",
          "steps": [
            {"seq": 1, "kind":"patrol",
             "enc": {
               "units":[{"unitId":"0999"}],
               "sets":[{"name":"Fernwood","kind":"city"}],
               "rsrc":[{"kind":"ore","qty":1}]
             }
            },
            {"seq": 2, "kind":"obs",
             "terr":"forest",
             "special": true,
             "label": "Ancient Ruins",
             "borders":[{"dir":"N","kind":"river"},{"dir":"NW","kind":"ford"}],
             "enc": { "units":[{"unitId":"0123"}] }
            }
          ]
        },
        {
          "seq": 5,
          "kind": "status",
          "steps": [
            {"seq": 1, "kind":"obs", "terr":"plains"}
          ]
        }
      ]
    }
  ]
}
```

### Discriminators

- `Act.kind`: `follow | goto | move | scout | status`
- `Step.kind`: `adv | still | patrol | obs`

The JSON uses discriminator fields (`kind`) and **optional** fields that correspond directly to nullable columns in `acts`/`steps` plus normalized child tables (`step_enc_*`, `step_borders`).

## Tile export/import

```json
{
  "hex": "+12-4-8",
  "terr": "forest",
  "specialLabel": "Ancient Ruins",
  "units": [{"unitId":"0123"}],
  "sets": [{"name":"Fernwood","kind":"city"}],
  "rsrc": [{"kind":"ore","qty":1}],
  "borders": [{"dir":"N","kind":"river"}],
  "src": [
    {"docId":77,"unitId":"0346","turnNo":90306,"actSeq":4,"stepSeq":2}
  ]
}
```

`Tile.src` is the merge-conflict tool: when two shared datasets disagree, you can show *which extracted evidence* produced each claim.
