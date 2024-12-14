# Benchmark "old retrypool vs revamp retrypool"

Clearly the revamp is a success in terms of performances.

## Old retrypool

```
% go run ./bench 
Running 10 benchmark iterations with 10000 inserts each...

Benchmark run 1/10:
  Total inserts: 10000
  Duration: 0.19 seconds
  Inserts/second: 52533.37
  Successful: 10000
  Failed: 0

Benchmark run 2/10:
  Total inserts: 10000
  Duration: 0.17 seconds
  Inserts/second: 59561.06
  Successful: 10000
  Failed: 0

Benchmark run 3/10:
  Total inserts: 10000
  Duration: 0.17 seconds
  Inserts/second: 59274.30
  Successful: 10000
  Failed: 0

Benchmark run 4/10:
  Total inserts: 10000
  Duration: 0.17 seconds
  Inserts/second: 58226.52
  Successful: 10000
  Failed: 0

Benchmark run 5/10:
  Total inserts: 10000
  Duration: 0.17 seconds
  Inserts/second: 58614.30
  Successful: 10000
  Failed: 0

Benchmark run 6/10:
  Total inserts: 10000
  Duration: 0.17 seconds
  Inserts/second: 57970.33
  Successful: 10000
  Failed: 0

Benchmark run 7/10:
  Total inserts: 10000
  Duration: 0.17 seconds
  Inserts/second: 59360.20
  Successful: 10000
  Failed: 0

Benchmark run 8/10:
  Total inserts: 10000
  Duration: 0.17 seconds
  Inserts/second: 57906.38
  Successful: 10000
  Failed: 0

Benchmark run 9/10:
  Total inserts: 10000
  Duration: 0.17 seconds
  Inserts/second: 59212.18
  Successful: 10000
  Failed: 0

Benchmark run 10/10:
  Total inserts: 10000
  Duration: 0.17 seconds
  Inserts/second: 59178.45
  Successful: 10000
  Failed: 0

Average inserts/second across 10 successful runs: 58183.71
```


## Revamp 

```
% go run ./bench
Running 10 benchmark iterations with 10000 inserts each...

Benchmark run 1/10:
  Total inserts: 10000
  Duration: 0.09 seconds
  Inserts/second: 105559.56
  Successful: 10000
  Failed: 0

Benchmark run 2/10:
  Total inserts: 10000
  Duration: 0.07 seconds
  Inserts/second: 144591.38
  Successful: 10000
  Failed: 0

Benchmark run 3/10:
  Total inserts: 10000
  Duration: 0.07 seconds
  Inserts/second: 143823.76
  Successful: 10000
  Failed: 0

Benchmark run 4/10:
  Total inserts: 10000
  Duration: 0.07 seconds
  Inserts/second: 147713.67
  Successful: 10000
  Failed: 0

Benchmark run 5/10:
  Total inserts: 10000
  Duration: 0.07 seconds
  Inserts/second: 144946.97
  Successful: 10000
  Failed: 0

Benchmark run 6/10:
  Total inserts: 10000
  Duration: 0.07 seconds
  Inserts/second: 142479.23
  Successful: 10000
  Failed: 0

Benchmark run 7/10:
  Total inserts: 10000
  Duration: 0.07 seconds
  Inserts/second: 143601.48
  Successful: 10000
  Failed: 0

Benchmark run 8/10:
  Total inserts: 10000
  Duration: 0.07 seconds
  Inserts/second: 142248.02
  Successful: 10000
  Failed: 0

Benchmark run 9/10:
  Total inserts: 10000
  Duration: 0.07 seconds
  Inserts/second: 145191.35
  Successful: 10000
  Failed: 0

Benchmark run 10/10:
  Total inserts: 10000
  Duration: 0.07 seconds
  Inserts/second: 140879.63
  Successful: 10000
  Failed: 0

Average inserts/second across 10 successful runs: 140103.50
```
