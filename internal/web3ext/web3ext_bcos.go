package web3ext

import "github.com/bcos-one/BCOS/params"

// TODO web3j
const BCOS_JS = `
web3._extend({
	property: '` + params.ClientIdentifier + `',
	methods: [
		new web3._extend.Method({
			name: 'chainId',
			call: '` + params.ClientIdentifier + `_chainId',
			params: 0
		}),
		new web3._extend.Method({
			name: 'sign',
			call: '` + params.ClientIdentifier + `_sign',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, null]
		}),
		new web3._extend.Method({
			name: 'resend',
			call: '` + params.ClientIdentifier + `_resend',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter, web3._extend.utils.fromDecimal, web3._extend.utils.fromDecimal]
		}),
		new web3._extend.Method({
			name: 'signTransaction',
			call: '` + params.ClientIdentifier + `_signTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
			name: 'submitTransaction',
			call: '` + params.ClientIdentifier + `_submitTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
			name: 'getRawTransaction',
			call: '` + params.ClientIdentifier + `_getRawTransactionByHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getRawTransactionFromBlock',
			call: function(args) {
				return (web3._extend.utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? '` + params.ClientIdentifier + `_getRawTransactionByBlockHashAndIndex' : '` + params.ClientIdentifier + `_getRawTransactionByBlockNumberAndIndex';
			},
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, web3._extend.utils.toHex]
		}),
		new web3._extend.Method({
			name: 'getProof',
			call: '` + params.ClientIdentifier + `_getProof',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, null, web3._extend.formatters.inputBlockNumberFormatter]
		}),
		//TODO
		new web3._extend.Method({
			name: 'getBalance',
			call: '` + params.ClientIdentifier + `_getBalance',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputBlockNumberFormatter]
			outputFormatter: [web3._extend.formatters.outputBigNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getStorageAt',
			call: '` + params.ClientIdentifier + `_getStorageAt',
			params: 3,
			inputFormatter: [null, web3._extend.utils.toHex, web3._extend.formatters.inputDefaultBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getCode',
			call: '` + params.ClientIdentifier + `_getCode',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputDefaultBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getBlock',
			call: function(args) {
				return (web3._extend.utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? '` + params.ClientIdentifier + `_getBlockByHash' : '` + params.ClientIdentifier + `_getBlockByNumber';
			},
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, function (val) {return !!val;}],
            outputFormatter: web3._extend.formatters.outputBlockFormatter
        }),
		new web3._extend.Method({
			name: 'getUncle',
			call: function(args) {
				return (web3._extend.utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? '` + params.ClientIdentifier + `_getUncleByBlockHashAndIndex' : '` + params.ClientIdentifier + `_getUncleByBlockNumberAndIndex';
			},
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, web3._extend.utils.toHex],
			outputFormatter: web3._extend.formatters.outputBlockFormatter,
        }),
		new web3._extend.Method({
			name: 'getCompilers',
			call: '` + params.ClientIdentifier + `_getCompilers',
			params: 0
        }),
		new web3._extend.Method({
			name: 'getBlockTransactionCount',
			//call: getBlockTransactionCountCall,
			call: function(args) {
				return (web3._extend.utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? '` + params.ClientIdentifier + `_getBlockTransactionCountByHash' : '` + params.ClientIdentifier + `_getBlockTransactionCountByNumber';
			},
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter],
			outputFormatter: web3._extend.utils.toDecimal
        }),
		new web3._extend.Method({
			name: 'getBlockUncleCount',
			//call: uncleCountCall,
			call: function(args) {
				return (web3._extend.utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? '` + params.ClientIdentifier + `_getUncleCountByBlockHash' : '` + params.ClientIdentifier + `_getUncleCountByBlockNumber';
			},
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter],
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Method({
			name: 'getTransaction',
			call: '` + params.ClientIdentifier + `getTransactionByHash',
			params: 1,
			outputFormatter: web3._extend.formatters.outputTransactionFormatter
        }),
		new web3._extend.Method({
			name: 'getTransactionFromBlock',
			//call: transactionFromBlockCall,
			call: function(args) {
				return (web3._extend.utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? '` + params.ClientIdentifier + `_getTransactionByBlockHashAndIndex' : '` + params.ClientIdentifier + `_getTransactionByBlockNumberAndIndex';
			},
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, web3._extend.utils.toHex],
			outputFormatter: web3._extend.formatters.outputTransactionFormatter
           }),
		new web3._extend.Method({
			name: 'getTransactionReceipt',
			call: '` + params.ClientIdentifier + `getTransactionReceipt',
			params: 1,
			outputFormatter: web3._extend.formatters.outputTransactionReceiptFormatter
		}),
		new web3._extend.Method({
			name: 'getTransactionCount',
			call: '` + params.ClientIdentifier + `getTransactionCount',
			params: 2,
			inputFormatter: [null, web3._extend.formatters.inputDefaultBlockNumberFormatter],
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Method({
			name: 'sendRawTransaction',
			call: '` + params.ClientIdentifier + `sendRawTransaction',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'sendTransaction',
			call: '` + params.ClientIdentifier + `sendTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
			name: 'signTransaction',
			call: '` + params.ClientIdentifier + `signTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
           name: 'sign',
			call: '` + params.ClientIdentifier + `sign',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, null]
		}),
		new web3._extend.Method({
			name: 'call',
			call: '` + params.ClientIdentifier + `call',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputCallFormatter, web3._extend.formatters.inputDefaultBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'estimateGas',
			call: '` + params.ClientIdentifier + `estimateGas',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputCallFormatter],
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Method({
			name: 'compile.solidity',
			call: '` + params.ClientIdentifier + `compileSolidity',
			params: 1
		}),
		new web3._extend.Method({
			name: 'compile.lll',
			call: '` + params.ClientIdentifier + `compileLLL',
			params: 1
		}),
		new web3._extend.Method({
			name: 'compile.serpent',
			call: '` + params.ClientIdentifier + `compileSerpent',
			params: 1
		}),
		new web3._extend.Method({
			name: 'submitWork',
			call: '` + params.ClientIdentifier + `submitWork',
			params: 3
		}),
		new web3._extend.Method({
			name: 'getWork',
			call: '` + params.ClientIdentifier + `getWork',
			params: 0
		}),

	],
	properties: [
		new web3._extend.Property({
			name: 'pendingTransactions',
			getter: '` + params.ClientIdentifier + `_pendingTransactions',
			outputFormatter: function(txs) {
				var formatted = [];
				for (var i = 0; i < txs.length; i++) {
					formatted.push(web3._extend.formatters.outputTransactionFormatter(txs[i]));
					formatted[i].blockHash = null;
				}
				return formatted;
			}
		}),
		//TODO
		new web3._extend.Property({
			name: 'coinbase',
			getter: '` + params.ClientIdentifier + `_coinbase'
		}),
		new web3._extend.Property({
			name: 'mining',
			getter: '` + params.ClientIdentifier + `_mining'
		}),
		new web3._extend.Property({
			name: 'hashrate',
			getter: '` + params.ClientIdentifier + `_hashrate',
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Property({
			name: 'syncing',
			getter: '` + params.ClientIdentifier + `_syncing',
			outputFormatter: web3._extend.formatters.outputSyncingFormatter
		}),
		new web3._extend.Property({
			name: 'gasPrice',
			getter: '` + params.ClientIdentifier + `_gasPrice',
			outputFormatter: web3._extend.formatters.outputBigNumberFormatter
		}),
		new web3._extend.Property({
			name: 'accounts',
			getter: '` + params.ClientIdentifier + `_accounts'
		}),
		new web3._extend.Property({
			name: 'blockNumber',
			getter: '` + params.ClientIdentifier + `_blockNumber',
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Property({
			name: 'protocolVersion',
			getter: '` + params.ClientIdentifier + `_protocolVersion'
		}),
	]
});
`
