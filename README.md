# go-relayer
Listens to blocks at the quark layer and submits data to the Avail DA

## Usage

### Prerequisites

To run the go relayer, make sure you spin up the quark layer and the sausage server on your local sysytem, to do so you can follow the instructions [here]([https://github.com/sausaging](https://github.com/sausaging#mvp-quick-start))

### Build Steps

1. Clone the repository:

   ```bash
   git clone https://github.com/sausaging/go-relayer.git
   cd go-relayer

2. Add configs
   
   ```bash
   cd ./dataSubmit && cp config.example.json config.json
Make sure you replace your seed phrase and the app id in the config file

3. Run go-relayer
   
   ```bash
   go run submit.go --config config.json
   


