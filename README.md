# go-daly-bms

Go porting of [python-daly-bms](https://github.com/dreadnought/python-daly-bms) library to interact with Daly BMS.


## Installation

```bash
go get github.com/jonamat/go-daly-bms
```

## Usage

```go
package main

import (
	"fmt"

	bms "github.com/jonamat/go-daly-bms/pkg/bms"
)

func main() {
	bms := bms.DalyBMS()
	if err := bms.Connect("/dev/ttyUSB0"); err != nil {
		panic(err)
	}
	defer bms.Disconnect()

	statusData, err := bms.GetStatus()
	if err != nil {
		panic(err)
	} 

	fmt.Printf("Cycles: %+v\n", statusData.CycleCount)
	

	socData, err := bms.GetSOC()
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("SOC Percent: %+v\n", socData.SOCPercent)
}
```

## License

MIT