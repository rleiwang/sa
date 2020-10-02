---
## BWT construction by inducing SA

```go

var text []byte
// fill data in buffer text
...

// transform text to BWT, cnt is length of original text
cnt, bwt, _ := BWT(text)

```

Please note, this implementation is different from others in following:
1. *sentinel* starts from the beginning of the text, ie, LMS is actually RMS.
2. only supports UTF-8 encoded text input
3. Multi strings use byte value (1) as divider


## References
This implementation has referenced the following papers and code

* [Inducing enhanced suffix arrays for string collections](https://www.sciencedirect.com/science/article/pii/S0304397517302621) F. A. Louza, S. Gog and G. P. Telles
* [Two Efficient Algorithms for Linear Time Suffix Array Construction](https://ieeexplore.ieee.org/abstract/document/5582081) Ge Nong, Sen Zhang, Wai Hong Chan
* [sais-lite](https://sites.google.com/site/yuta256/) Yuta Mori