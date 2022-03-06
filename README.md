# CS-438 Project: GossipNet

Authors: Wanhao Zhou, Victor Carles, and Martin Beaussart

## Run the code
To start the backend,
```bash
cd ./gui && go run mod.go start
```

To start the frontend, simply open the `./gui/web/index.html` and type the corresponding node socket address.

## Run the static code analysis
```bash
make lint
```

## Run the test
We include four test suites in our code as stated in the report. To run all tests,
```bash
make test
```