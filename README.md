# payment-system
simple payments API  

to run see Makefile

### Methods:
##### create a wallet
```shell
curl 'http://0.0.0.0:3000/v1/createWallet?wallet=66fd0095-1dc2-4064-835f-1a2c24a29581'
```
response:
```json
{"data":"ok","code":200}
```

##### get a wallet
```shell
curl 'http://0.0.0.0:3000/v1/getWallet?wallet=66fd0095-1dc2-4064-835f-1a2c24a29581'
```
response:
```json
{
  "data": {
    "amount": 0,
    "wallet": "66fd0095-1dc2-4064-835f-1a2c24a29581",
    "owner": 0,
    "status": 0,
    "updated": "2021-08-19T16:38:26.61599Z",
    "created": "2021-08-19T16:38:26.61599Z"
  },
  "code": 200
}
```
##### deposit to a wallet
requires a unique transaction key
```shell
curl 'http://0.0.0.0:3000/v1/deposit?wallet=66fd0095-1dc2-4064-835f-1a2c24a29581&amount=100&key=4'
```
response:
```json
{"data":"ok","code":200}
```
##### withdraw from a wallet
requires a unique transaction key
```shell
curl 'http://0.0.0.0:3000/v1/withdraw?wallet=66fd0095-1dc2-4064-835f-1a2c24a29581&amount=20.5&key=5'
```
response:
```json
{"data":"ok","code":200}
```
##### transfer funds from a wallet to another
requires a unique transaction key
```shell
curl 'http://0.0.0.0:3000/v1/transferFunds?from=66fd0095-1dc2-4064-835f-1a2c24a29580&to=66fd0095-1dc2-4064-835f-1a2c24a29581&amount=40&key=7'
```
response:
```json
{"data":"ok","code":200}
```
##### create a report
returns list of transactions
transaction types:
- 0 or deposit: deposit
- 1 or withdraw or withdrawal: withdraw
- 2 or transfer or transferfrom: transfers from specified wallet
- 3 or transferto: transfers to specified wallet
- -1 or  no type: all transactions
```shell
curl 'http://0.0.0.0:3000/v1/report?wallet=66fd0095-1dc2-4064-835f-1a2c24a29581&from=2021-08-20&to2021-09-13&type=0'
```
response:
```json
{
  "data": [
    {
      "id": 1,
      "type": 0,
      "wallet": "66fd0095-1dc2-4064-835f-1a2c24a29581",
      "wallet_receiver": "",
      "key": "4",
      "amount": 100,
      "ts": "2021-08-19T14:11:39.960323Z"
    }
  ],
  "code": 200
}
```
