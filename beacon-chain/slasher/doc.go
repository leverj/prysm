// nolint:dupword
/*
Package slasher defines an optimized implementation of Ethereum proof-of-stake slashing
detection, namely focused on catching "surround vote" slashable
offenses as explained here: https://blog.ethereum.org/2020/01/13/validated-staking-on-eth2-1-incentives/.

Surround vote detection is a difficult problem if done naively, as slasher
needs to keep track of every single attestation by every single validator
in the network and be ready to efficiently detect whether incoming attestations
are slashable with respect to older ones. To do this, the Sigma Prime team
created an elaborate design document: https://hackmd.io/@sproul/min-max-slasher
offering an optimal solution.

Attesting histories are kept for each validator in two separate arrays known
as min and max spans, which are explained in our design document:
https://hackmd.io/@prysmaticlabs/slasher.

This is known as 2D chunking, pioneered by the Sigma Prime team here:
https://hackmd.io/@sproul/min-max-slasher. The parameters H, C, and K will be
used extensively throughout this package.

Attestations are represented as following: `<source epoch>====><target epoch>`
N: Number of epochs worth of history we want to keep for each validator.

In the following example:
- N = 4096
- Validators 257 and 258 have some attestations
- All other validators have no attestations

For MIN SPAN, `∞“ is actually set to the max `uint16` value: 65535

	validator   257 :       8193=======>8195  8196=>8197=============>8200                    8204=>8205=>8206=>8207=========>8209=>8210=>8211=>8212=>8213=>8214                          8219=>8220  8221=>8222
	validator   258 :             8193=======>8196=>8197=>8198=>8199=>8200=>8201=======>8203=>8204=>8205=>8206=>8207===>8208=>8209=>8210=>8211=>8212=>8213=>8214=>8215=>8216=>8217=>8218=>8219=>8220=>8221

/----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------\
| MIN SPAN          | kN+0  kN+1  kN+2  kN+3  kN+4  kN+5  kN+6  kN+7  kN+8  kN+9  kN+10 kN+11 kN+12 kN+13 kN+14 kN+15 | kN+16 kN+17 kN+18 kN+19 kN+20 kN+21 kN+22 kN+23 kN+24 kN+25 kN+26 kN+27 kN+28 kN+29 kN+30 kN+31 | ... | (k+1)N-16 (k+1)N-15 (k+1)N-14 (k+1)N-13 (k+1)N-12 (k+1)N-11 (k+1)N-10 (k+1)N-9  (k+1)N-8  (k+1)N-7  (k+1)N-6  (k+1)N-5  (k+1)N-4  (k+1)N-3  (k+1)N-2  (k+1)N-1 |
|-------------------+-------------------------------------------------------------------------------------------------+-------------------------------------------------------------------------------------------------+-----+----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| validator       0 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
| validator       1 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
| validator       2 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
| ................. | ............................................................................................... | ............................................................................................... | ... | .............................................................................................................................................................. |
| validator     254 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
| validator     255 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
|-------------------+-------------------------------------------------------------------------------------------------+-------------------------------------------------------------------------------------------------+-----+----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| validator     256 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
| validator     257 |   3     4     3     2     4     8     7     6     5     4     3     2     2     2     3     3   |   2     2     2     2     2     7     6     5     4     3     2     3     2     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
| validator     258 |   4     3     3     2     2     2     2     2     3     3     2     2     2     2     2     2   |   2     2     2     2     2     2     2     2     2     2     2     2     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
| ................. | ............................................................................................... | ............................................................................................... | ... | .............................................................................................................................................................. |
| validator     510 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
| validator     511 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
|-------------------+-------------------------------------------------------------------------------------------------+-------------------------------------------------------------------------------------------------+-----+----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ................. | ............................................................................................... | ............................................................................................... | ... | .............................................................................................................................................................. |
|-------------------+-------------------------------------------------------------------------------------------------+-------------------------------------------------------------------------------------------------+-----+----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| validator M - 256 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
| validator M - 255 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
| ................. | ............................................................................................... | ............................................................................................... | ... | .............................................................................................................................................................. |
| validator M -   1 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   | ... |       ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞         ∞  |
\----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------/

/----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------\
| MAX SPAN          | kN+0  kN+1  kN+2  kN+3  kN+4  kN+5  kN+6  kN+7  kN+8  kN+9  kN+10 kN+11 kN+12 kN+13 kN+14 kN+15 | kN+16 kN+17 kN+18 kN+19 kN+20 kN+21 kN+22 kN+23 kN+24 kN+25 kN+26 kN+27 kN+28 kN+29 kN+30 kN+31 | ... | (k+1)N-16 (k+1)N-15 (k+1)N-14 (k+1)N-13 (k+1)N-12 (k+1)N-11 (k+1)N-10 (k+1)N-9  (k+1)N-8  (k+1)N-7  (k+1)N-6  (k+1)N-5  (k+1)N-4  (k+1)N-3  (k+1)N-2  (k+1)N-1 |
|-------------------+-------------------------------------------------------------------------------------------------+-------------------------------------------------------------------------------------------------+-----+----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| validator       0 |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| validator       1 |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| validator       2 |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| ................. | ............................................................................................... | ............................................................................................... | ... | .............................................................................................................................................................. |
| validator      14 |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| validator      15 |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| ------------------+-------------------------------------------------------------------------------------------------+-------------------------------------------------------------------------------------------------+-----+----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| validator     256 |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| validator     257 |   0     0     1     0     0     0     2     1     0     0     0     0     0     0     0     0   |   1     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| validator     258 |   0     0     0     1     0     0     0     0     0     0     1     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| ................. | ............................................................................................... | ............................................................................................... | ... | .............................................................................................................................................................. |
| validator     510 |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| validator     511 |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| ------------------+-------------------------------------------------------------------------------------------------+-------------------------------------------------------------------------------------------------+-----+----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ................. | ............................................................................................... | ............................................................................................... | ... | .............................................................................................................................................................. |
| ------------------+-------------------------------------------------------------------------------------------------+-------------------------------------------------------------------------------------------------+-----+----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| validator M - 256 |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| validator M - 255 |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
| ................. | ............................................................................................... | ............................................................................................... | ... | .............................................................................................................................................................. |
| validator M -   1 |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   |   0     0     0     0     0     0     0     0     0     0     0     0     0     0     0     0   | ... |       0         0         0         0         0         0         0         0         0         0         0         0         0         0         0         0  |
\ ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------/

How to know if an incoming attestation will surround a pre-existing one?
------------------------------------------------------------------------
Example with an incoming attestation 8197====>8199 for validator 257.
- First, we retrieve the MIN SPAN value for the source epoch of the incoming attestation (here the source epoch is 8197). We get the value 8.
- Then, for the incoming attestation, we compute `target - source`. We get the value 8199 - 8197 = 2.
- 8 >= 2, so the incoming attestation will NOT surround any pre-existing one.

Example with an incoming attestation 8202====>8206 for validator 257.
- First, we retrieve the MIN SPAN value for the source epoch of the incoming attestation (here the source epoch is 8202). We get the value 3.
- Then, for the incoming attestation, we compute `target - source`. We get the value 8206 - 8202 = 4.
- 3 < 4, so the incoming attestation will surround a pre-existing one. (In this precise case, it will surround 8204=>8205)

How to know if an incoming attestation will be surrounded by a pre-existing one?
--------------------------------------------------------------------------------
Example with an incoming attestation 8197====>8199 for validator 257.
- First, we retrieve the MAX SPAN value for the source epoch of the incoming attestation (here the source epoch is 8197). We get the value 0.
- Then, for the incoming attestation, we compute `target - source`. We get the value 8199 - 8197 = 2.
- 0 <= 2, so the incoming attestation will NOT be surrounded by any pre-existing one.

Example with an incoming attestation 8198====>8199 for validator 257.
- First, we retrieve the MAX SPAN value for the source epoch of the incoming attestation (here the source epoch is 8198). We get the value 2.
- Then, for the incoming attestation, we compute `target - source`. We get the value 8199 - 8198 = 1.
- 2 > 1, so the incoming attestation will be surrounded by a pre-existing one. (In this precise case, it will be surrounded by 8197=>8200)

Data are stored on disk by chunk.
For example: For MIN SPAN, validators 256 to 511 included, epochs 8208 to 8223 included, the corresponding chunk is:
/---------------------------------------------------------------------------------------------------------------------\
| MIN SPAN          | kN+16 kN+17 kN+18 kN+19 kN+20 kN+21 kN+22 kN+23 kN+24 kN+25 kN+26 kN+27 kN+28 kN+29 kN+30 kN+31 |
|-------------------+-------------------------------------------------------------------------------------------------|
| validator     256 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |
| validator     257 |   2     2     2     2     2     7     6     5     4     3     2     3     2     ∞     ∞     ∞   |
| validator     258 |   2     2     2     2     2     2     2     2     2     2     2     2     ∞     ∞     ∞     ∞   |
| ................. | ............................................................................................... |
| validator     510 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |
| validator     511 |   ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞     ∞   |
\---------------------------------------------------------------------------------------------------------------------/

Chunks are stored into the database a flat array of bytes.
For this example, the stored value will be:
|          validator 256        |           validator 257       |        validator 258          |...|          validator 510        |         validator 511		    |
[∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,2,2,2,2,2,7,6,5,4,3,2,3,2,∞,∞,∞,2,2,2,2,2,2,2,2,2,2,2,2,∞,∞,∞,∞,...,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞,∞]

A chunk contains 256 validators * 16 epochs = 4096 values.
A chunk value is stored on 2 bytes (uint16).
==> A chunk takes 8192 bytes = 8KB

There is 4096 epochs / 16 epochs per chunk = 256 chunks per batch of 256 validators.
Storing all values fo a batch of 256 validators takes 256 * 8KB = 2MB
With 1_048_576 validators, we need 4096 * 2MB = 8GB

Storing both MIN and MAX spans for 1_048_576 validators takes 16GB.

Each chunk is stored snappy-compressed in the database.
If all validators attest ideally, a MIN SPAN chunk will contain only `2`s, and MAX SPAN chunk will contain only `0`s.
This will compress very well, and will let us store a lot of data in a small amount of space.
*/

package slasher
