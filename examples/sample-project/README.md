# Sample project

A small repository that shows what ShipProof produces. The evidence workflow
runs the action against this directory on every pull request, and the run
uploads the evidence pack it builds.

The change `SP-1` is open. It holds one requirement, one proof that runs, and
the code the proof judges.

Run it yourself:

```bash
cd examples/sample-project
shipproof pack SP-1
shipproof status SP-1
```
