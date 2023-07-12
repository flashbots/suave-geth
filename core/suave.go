package core

const BidsContractCode string = "0x6080604052600436106100345760003560e01c8063274e3b04146100995780633c34db65146100c257806392f07a5814610131575b60405162461bcd60e51b815260206004820152603560248201527f436f6e747261637420646f6573206e6f7420686176652066616c6c6261636b2060448201908152746e6f7220726563656976652066756e6374696f6e7360581b6064830152608482fd5b6100ac6100a736600461098c565b610193565b6040516100b99190610a9b565b60405180910390f35b34801561011b5760405162461bcd60e51b815260206004820152602260248201527f45746865722073656e7420746f206e6f6e2d70617961626c652066756e637469604482019081526137b760f11b6064830152608482fd5b5061012f61012a366004610ab5565b61040e565b005b34801561018a5760405162461bcd60e51b815260206004820152602260248201527f45746865722073656e7420746f206e6f6e2d70617961626c652066756e637469604482019081526137b760f11b6064830152608482fd5b506100ac610474565b606061019d61049c565b6101a657600080fd5b6000306001600160a01b03166392f07a586040518163ffffffff1660e01b8152600401600060405180830381600087803b1580156102335760405162461bcd60e51b815260206004820152602560248201527f54617267657420636f6e747261637420646f6573206e6f7420636f6e7461696e604482019081526420636f646560d81b6064830152608482fd5b505af1158015610247573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f1916820160405261026f9190810190610b45565b905060008061027d83610500565b91509150816102de5760405162461bcd60e51b815260206004820152602260248201527f62756e646c6520646f6573206e6f742073696d756c61746520636f72726563746044820152616c7960f01b60648201526084015b60405180910390fd5b60006102ea8787610587565b905061031c81600001516040518060400160405280600981526020016865746842756e646c6560b81b81525086610669565b8051604080518082018252601381527265746842756e646c6553696d526573756c747360681b60208083019190915282516001600160401b038716818301528351808203909201825283019092526103749291610669565b7f83481d5b04dea534715acad673a8177a46fc93882760f36bdc16ccac439d504e8160000151826020015183604001516040516103b393929190610c73565b60405180910390a1604051633c34db6560e01b906103d5908390602001610ca5565b60408051601f19818403018152908290526103f39291602001610d25565b60405160208183030381529060405294505050505092915050565b7f83481d5b04dea534715acad673a8177a46fc93882760f36bdc16ccac439d504e61043c6020830183610d6c565b61044c6040840160208501610d8c565b6104596040850185610df1565b6040516104699493929190610ec4565b60405180910390a150565b6060600061048061071b565b9050808060200190518101906104969190610b45565b91505090565b6040516000908190819063420100009082818181855afa9150503d80600081146104e2576040519150601f19603f3d011682016040523d82523d6000602084013e6104e7565b606091505b5091509150816104f657600080fd5b6020015192915050565b60008060008063421000006001600160a01b0316856040516105229190610f39565b600060405180830381855afa9150503d806000811461055d576040519150601f19603f3d011682016040523d82523d6000602084013e610562565b606091505b5091509150818180602001905181019061057c9190610f65565b935093505050915091565b6040805160608082018352600080835260208301529181019190915260008063420300006001600160a01b031685856040516020016105c7929190610f85565b60408051601f19818403018152908290526105e191610f39565b600060405180830381855afa9150503d806000811461061c576040519150601f19603f3d011682016040523d82523d6000602084013e610621565b606091505b50915091508161064c576342030000816040516375fff46760e01b81526004016102d5929190610faf565b808060200190518101906106609190611099565b95945050505050565b60008063420200006001600160a01b031685858560405160200161068f9392919061117c565b60408051601f19818403018152908290526106a991610f39565b600060405180830381855afa9150503d80600081146106e4576040519150601f19603f3d011682016040523d82523d6000602084013e6106e9565b606091505b509150915081610714576342020000816040516375fff46760e01b81526004016102d5929190610faf565b5050505050565b604051606090600090819063420100019082818181855afa9150503d8060008114610762576040519150601f19603f3d011682016040523d82523d6000602084013e610767565b606091505b50915091508161077657600080fd5b92915050565b60405162461bcd60e51b815260206004820152602260248201527f414249206465636f64696e673a207475706c65206461746120746f6f2073686f6044820152611c9d60f21b6064820152608481fd5b60405162461bcd60e51b815260206004820152602260248201527f414249206465636f64696e673a20696e76616c6964207475706c65206f666673604482015261195d60f21b6064820152608481fd5b6001600160401b038116811461083157600080fd5b50565b60405162461bcd60e51b815260206004820152602b60248201527f414249206465636f64696e673a20696e76616c69642063616c6c64617461206160448201526a1c9c985e481bd9999cd95d60aa1b6064820152608481fd5b634e487b7160e01b600052604160045260246000fd5b604051606081016001600160401b03811182821017156108c5576108c561088d565b60405290565b604051601f8201601f191681016001600160401b03811182821017156108f3576108f361088d565b604052919050565b60006001600160401b038211156109145761091461088d565b5060051b60200190565b60405162461bcd60e51b815260206004820152602b60248201527f414249206465636f64696e673a20696e76616c69642063616c6c64617461206160448201526a727261792073747269646560a81b6064820152608481fd5b6001600160a01b038116811461083157600080fd5b600080604083850312156109a2576109a261077c565b82356109ad8161081c565b91506020838101356001600160401b038111156109cc576109cc6107cc565b8401601f810186136109e0576109e0610834565b80356109f36109ee826108fb565b6108cb565b81815260059190911b82018301908381019088831115610a1557610a1561091e565b928401925b82841015610a3c578335610a2d81610977565b82529284019290840190610a1a565b80955050505050509250929050565b60005b83811015610a66578181015183820152602001610a4e565b50506000910152565b60008151808452610a87816020860160208601610a4b565b601f01601f19169290920160200192915050565b602081526000610aae6020830184610a6f565b9392505050565b600060208284031215610aca57610aca61077c565b81356001600160401b03811115610ae357610ae36107cc565b820160608185031215610aae5760405162461bcd60e51b815260206004820152602760248201527f414249206465636f64696e673a207374727563742063616c6c6461746120746f6044820152661bc81cda1bdc9d60ca1b6064820152608481fd5b60006020808385031215610b5b57610b5b61077c565b82516001600160401b0380821115610b7557610b756107cc565b818501915085601f830112610b8c57610b8c610834565b815181811115610b9e57610b9e61088d565b610bb0601f8201601f191685016108cb565b91508082528684828501011115610c165760405162461bcd60e51b815260048101859052602760248201527f414249206465636f64696e673a20696e76616c69642062797465206172726179604482015266040d8cadccee8d60cb1b6064820152608481fd5b610c2581858401868601610a4b565b5095945050505050565b600081518084526020808501945080840160005b83811015610c685781516001600160a01b031687529582019590820190600101610c43565b509495945050505050565b6001600160801b0319841681526001600160401b03831660208201526060604082015260006106606060830184610c2f565b602080825282516001600160801b03191682820152828101516001600160401b031660408084019190915283015160608084015280516080840181905260009291820190839060a08601905b80831015610d1a5783516001600160a01b03168252928401926001929092019190840190610cf1565b509695505050505050565b6001600160e01b0319831681528151600090610d48816004850160208701610a4b565b919091016004019392505050565b6001600160801b03198116811461083157600080fd5b600060208284031215610d8157610d8161077c565b8135610aae81610d56565b600060208284031215610da157610da161077c565b8135610aae8161081c565b60405162461bcd60e51b815260206004820152601760248201527f43616c6c64617461207461696c20746f6f2073686f72740000000000000000006044820152606481fd5b6000808335601e19843603018112610e485760405162461bcd60e51b815260206004820152601c60248201527f496e76616c69642063616c6c64617461207461696c206f6666736574000000006044820152606481fd5b8301803591506001600160401b03821115610ea25760405162461bcd60e51b815260206004820152601c60248201527f496e76616c69642063616c6c64617461207461696c206c656e677468000000006044820152606481fd5b6020019150600581901b3603821315610ebd57610ebd610dac565b9250929050565b6000606082016001600160801b03198716835260206001600160401b03871681850152606060408501528185835260808501905086925060005b86811015610f2c578335610f1181610977565b6001600160a01b031682529282019290820190600101610efe565b5098975050505050505050565b60008251610f4b818460208701610a4b565b9190910192915050565b8051610f608161081c565b919050565b600060208284031215610f7a57610f7a61077c565b8151610aae8161081c565b6001600160401b0383168152604060208201526000610fa76040830184610c2f565b949350505050565b6001600160a01b0383168152604060208201819052600090610fa790830184610a6f565b60405162461bcd60e51b815260206004820152602360248201527f414249206465636f64696e673a20696e76616c696420737472756374206f66666044820152621cd95d60ea1b6064820152608481fd5b8051610f6081610d56565b600082601f83011261104357611043610834565b815160206110536109ee836108fb565b82815260059290921b840181019181810190868411156110755761107561091e565b8286015b84811015610d1a57805161108c81610977565b8352918301918301611079565b6000602082840312156110ae576110ae61077c565b81516001600160401b03808211156110c8576110c86107cc565b90830190606082860312156111285760405162461bcd60e51b815260206004820152602360248201527f414249206465636f64696e673a20737472756374206461746120746f6f2073686044820152621bdc9d60ea1b6064820152608481fd5b6111306108a3565b61113983611024565b815261114760208401610f55565b602082015260408301518281111561116157611161610fd3565b61116d8782860161102f565b60408301525095945050505050565b6001600160801b03198416815260606020820152600061119f6060830185610a6f565b82810360408401526111b18185610a6f565b969550505050505056fea26469706673582212209c854677bba4ead2236c8f85837d0158d1e6b0b032c0b07a8069c159e451f55064736f6c63430008130033"

const BlockBidContractCode string = "0x608060405234801561005d5760405162461bcd60e51b815260206004820152602260248201527f45746865722073656e7420746f206e6f6e2d70617961626c652066756e637469604482019081526137b760f11b6064830152608482fd5b50600436106100a45760003560e01c80633c34db651461010957806346fda9fc1461011e57806374fa654f146101475780637df1cde21461015a578063e88473441461016d575b60405162461bcd60e51b815260206004820152603560248201527f436f6e747261637420646f6573206e6f7420686176652066616c6c6261636b2060448201908152746e6f7220726563656976652066756e6374696f6e7360581b6064830152608482fd5b61011c610117366004610d6c565b61018e565b005b61013161012c3660046111c6565b6101f4565b60405161013e919061126d565b60405180910390f35b610131610155366004611296565b6105ff565b61013161016836600461144d565b61080f565b61018061017b3660046114a2565b61084b565b60405161013e929190611605565b7f83481d5b04dea534715acad673a8177a46fc93882760f36bdc16ccac439d504e6101bc602083018361165c565b6101cc604084016020850161167c565b6101d960408501856116e1565b6040516101e994939291906117ad565b60405180910390a150565b60606000610201836108fa565b9050600081516001600160401b0381111561021e5761021e610e54565b60405190808252806020026020018201604052801561026357816020015b604080518082019091526000808252602082015281526020019060019003908161023c5790505b50905060005b82518110156103575760006102c684838151811061028957610289611822565b6020026020010151600001516040518060400160405280601381526020017265746842756e646c6553696d526573756c747360681b8152506109c9565b90506000818060200190518101906102de9190611838565b90506040518060400160405280826001600160401b0316815260200186858151811061030c5761030c611822565b6020026020010151600001516001600160801b03191681525084848151811061033757610337611822565b60200260200101819052505050808061034f9061186e565b915050610269565b50805160005b610368600183611887565b81101561047557600061037c82600161189a565b90505b828110156104625783818151811061039957610399611822565b6020026020010151600001516001600160401b03168483815181106103c0576103c0611822565b6020026020010151600001516001600160401b031611156104505760008483815181106103ef576103ef611822565b6020026020010151905084828151811061040b5761040b611822565b602002602001015185848151811061042557610425611822565b60200260200101819052508085838151811061044357610443611822565b6020026020010181905250505b8061045a8161186e565b91505061037f565b508061046d8161186e565b91505061035d565b50600083516001600160401b0381111561049157610491610e54565b6040519080825280602002602001820160405280156104ba578160200160208202803683370190505b50905060005b8351811015610524578381815181106104db576104db611822565b6020026020010151602001518282815181106104f9576104f9611822565b6001600160801b0319909216602092830291909101909101528061051c8161186e565b9150506104c0565b506040516374fa654f60e01b815230906374fa654f9061054c908a908a9086906004016119a7565b600060405180830381600087803b1580156105b65760405162461bcd60e51b815260206004820152602560248201527f54617267657420636f6e747261637420646f6573206e6f7420636f6e7461696e604482019081526420636f646560d81b6064830152608482fd5b505af11580156105ca573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f191682016040526105f29190810190611a30565b9450505050505b92915050565b60408051600280825260608083018452926000929190602083019080368337019050509050308160008151811061063857610638611822565b60200260200101906001600160a01b031690816001600160a01b03168152505063421000018160018151811061067057610670611822565b60200260200101906001600160a01b031690816001600160a01b031681525050600061069c8583610a74565b90506106ee81600001516040518060400160405280600a8152602001696d65726765644269647360b01b815250866040516020016106da9190611a6a565b604051602081830303815290604052610b4d565b6000806106ff888460000151610bff565b9150915061073883600001516040518060400160405280600e81526020016d189d5a5b19195c94185e5b1bd85960921b81525083610b4d565b82516040517f67fa9c16cd72410c4cc1d47205b31852a81ec5e92d1c8cebc3ecbe98ed67fe3f9161076a918590611a7d565b60405180910390a17f83481d5b04dea534715acad673a8177a46fc93882760f36bdc16ccac439d504e8360000151846020015185604001516040516107b193929190611aa0565b60405180910390a1604051633a211cd160e21b906107d59085908590602001611605565b60408051601f19818403018152908290526107f39291602001611ad2565b6040516020818303038152906040529450505050509392505050565b60606000610843846040518060400160405280600e81526020016d189d5a5b19195c94185e5b1bd85960921b8152506109c9565b949350505050565b6040805160608082018352600080835260208301529181019190915260607f67fa9c16cd72410c4cc1d47205b31852a81ec5e92d1c8cebc3ecbe98ed67fe3f84600001518460405161089e929190611a7d565b60405180910390a17f83481d5b04dea534715acad673a8177a46fc93882760f36bdc16ccac439d504e8460000151856020015186604001516040516108e593929190611aa0565b60405180910390a150829050815b9250929050565b604080516001600160401b038316602082015260609160009182916342030001910160408051601f198184030181529082905261093691611b03565b600060405180830381855afa9150503d8060008114610971576040519150601f19603f3d011682016040523d82523d6000602084013e610976565b606091505b5091509150816109aa576342030001816040516375fff46760e01b81526004016109a1929190611b1f565b60405180910390fd5b6000818060200190518101906109c09190611c19565b95945050505050565b606060008063420200016001600160a01b031685856040516020016109ef929190611a7d565b60408051601f1981840301815290829052610a0991611b03565b600060405180830381855afa9150503d8060008114610a44576040519150601f19603f3d011682016040523d82523d6000602084013e610a49565b606091505b509150915081610843576342020001816040516375fff46760e01b81526004016109a1929190611b1f565b6040805160608082018352600080835260208301529181019190915260008063420300006001600160a01b03168585604051602001610ab4929190611cca565b60408051601f1981840301815290829052610ace91611b03565b600060405180830381855afa9150503d8060008114610b09576040519150601f19603f3d011682016040523d82523d6000602084013e610b0e565b606091505b509150915081610b39576342030000816040516375fff46760e01b81526004016109a1929190611b1f565b808060200190518101906109c09190611cec565b60008063420200006001600160a01b0316858585604051602001610b7393929190611d26565b60408051601f1981840301815290829052610b8d91611b03565b600060405180830381855afa9150503d8060008114610bc8576040519150601f19603f3d011682016040523d82523d6000602084013e610bcd565b606091505b509150915081610bf8576342020000816040516375fff46760e01b81526004016109a1929190611b1f565b5050505050565b60608060008063421000016001600160a01b03168686604051602001610c26929190611d5b565b60408051601f1981840301815290829052610c4091611b03565b600060405180830381855afa9150503d8060008114610c7b576040519150601f19603f3d011682016040523d82523d6000602084013e610c80565b606091505b509150915081610cab576342100001816040516375fff46760e01b81526004016109a1929190611b1f565b80806020019051810190610cbf9190611d87565b9350935050509250929050565b60405162461bcd60e51b815260206004820152602260248201527f414249206465636f64696e673a207475706c65206461746120746f6f2073686f6044820152611c9d60f21b6064820152608481fd5b60405162461bcd60e51b815260206004820152602260248201527f414249206465636f64696e673a20696e76616c6964207475706c65206f666673604482015261195d60f21b6064820152608481fd5b600060208284031215610d8157610d81610ccc565b81356001600160401b03811115610d9a57610d9a610d1c565b820160608185031215610dfc5760405162461bcd60e51b815260206004820152602760248201527f414249206465636f64696e673a207374727563742063616c6c6461746120746f6044820152661bc81cda1bdc9d60ca1b6064820152608481fd5b9392505050565b60405162461bcd60e51b815260206004820152602360248201527f414249206465636f64696e673a20737472756374206461746120746f6f2073686044820152621bdc9d60ea1b6064820152608481fd5b634e487b7160e01b600052604160045260246000fd5b604051608081016001600160401b0381118282101715610e8c57610e8c610e54565b60405290565b604051606081016001600160401b0381118282101715610e8c57610e8c610e54565b604051601f8201601f191681016001600160401b0381118282101715610edc57610edc610e54565b604052919050565b60405162461bcd60e51b815260206004820152602360248201527f414249206465636f64696e673a20696e76616c696420737472756374206f66666044820152621cd95d60ea1b6064820152608481fd5b6001600160401b0381168114610f4a57600080fd5b50565b6001600160a01b0381168114610f4a57600080fd5b60405162461bcd60e51b815260206004820152602b60248201527f414249206465636f64696e673a20696e76616c69642063616c6c64617461206160448201526a1c9c985e481bd9999cd95d60aa1b6064820152608481fd5b60006001600160401b03821115610fd457610fd4610e54565b5060051b60200190565b60405162461bcd60e51b815260206004820152602b60248201527f414249206465636f64696e673a20696e76616c69642063616c6c64617461206160448201526a727261792073747269646560a81b6064820152608481fd5b600082601f83011261104b5761104b610f62565b8135602061106061105b83610fbb565b610eb4565b82815260079290921b8401810191818101908684111561108257611082610fde565b8286015b848110156110fb57608081890312156110a1576110a1610e03565b6110a9610e6a565b81356110b481610f35565b8152818501356110c381610f35565b818601526040828101356110d681610f4d565b908201526060828101356110e981610f35565b90820152835291830191608001611086565b509695505050505050565b600060c0828403121561111b5761111b610e03565b60405160c081016001600160401b03828210818311171561113e5761113e610e54565b81604052829350843583526020850135915061115982610f35565b8160208401526040850135915061116f82610f4d565b8160408401526060850135915061118582610f35565b8160608401526080850135608084015260a08501359150808211156111ac576111ac610ee4565b506111b985828601611037565b60a0830152505092915050565b600080604083850312156111dc576111dc610ccc565b82356001600160401b038111156111f5576111f5610d1c565b61120185828601611106565b925050602083013561121281610f35565b809150509250929050565b60005b83811015611238578181015183820152602001611220565b50506000910152565b6000815180845261125981602086016020860161121d565b601f01601f19169290920160200192915050565b602081526000610dfc6020830184611241565b6001600160801b031981168114610f4a57600080fd5b6000806000606084860312156112ae576112ae610ccc565b83356001600160401b03808211156112c8576112c8610d1c565b6112d487838801611106565b945060209150818601356112e781610f35565b93506040860135818111156112fe576112fe610d1c565b86019050601f8101871361131457611314610f62565b803561132261105b82610fbb565b81815260059190911b8201830190838101908983111561134457611344610fde565b928401925b8284101561136b57833561135c81611280565b82529284019290840190611349565b80955050505050509250925092565b60405162461bcd60e51b815260206004820152602760248201527f414249206465636f64696e673a20696e76616c69642062797465206172726179604482015266040d8cadccee8d60cb1b6064820152608481fd5b60006001600160401b038211156113e8576113e8610e54565b50601f01601f191660200190565b600082601f83011261140a5761140a610f62565b813561141861105b826113cf565b8181528460208386010111156114305761143061137a565b816020850160208301376000918101602001919091529392505050565b6000806040838503121561146357611463610ccc565b823561146e81611280565b915060208301356001600160401b0381111561148c5761148c610d1c565b611498858286016113f6565b9150509250929050565b600080604083850312156114b8576114b8610ccc565b82356001600160401b03808211156114d2576114d2610d1c565b90840190606082870312156114e9576114e9610e03565b6114f1610e92565b82356114fc81611280565b815260208381013561150d81610f35565b8282015260408401358381111561152657611526610ee4565b80850194505087601f85011261153e5761153e610f62565b833561154c61105b82610fbb565b81815260059190911b8501820190828101908a83111561156e5761156e610fde565b958301955b8287101561159557863561158681610f4d565b82529583019590830190611573565b604085015250919550860135925050808211156115b4576115b4610d1c565b50611498858286016113f6565b600081518084526020808501945080840160005b838110156115fa5781516001600160a01b0316875295820195908201906001016115d5565b509495945050505050565b604081526001600160801b031983511660408201526001600160401b036020840151166060820152600060408401516060608084015261164860a08401826115c1565b905082810360208401526109c08185611241565b60006020828403121561167157611671610ccc565b8135610dfc81611280565b60006020828403121561169157611691610ccc565b8135610dfc81610f35565b60405162461bcd60e51b815260206004820152601760248201527f43616c6c64617461207461696c20746f6f2073686f72740000000000000000006044820152606481fd5b6000808335601e198436030181126117385760405162461bcd60e51b815260206004820152601c60248201527f496e76616c69642063616c6c64617461207461696c206f6666736574000000006044820152606481fd5b8301803591506001600160401b038211156117925760405162461bcd60e51b815260206004820152601c60248201527f496e76616c69642063616c6c64617461207461696c206c656e677468000000006044820152606481fd5b6020019150600581901b36038213156108f3576108f361169c565b6000606082016001600160801b03198716835260206001600160401b03871681850152606060408501528185835260808501905086925060005b868110156118155783356117fa81610f4d565b6001600160a01b0316825292820192908201906001016117e7565b5098975050505050505050565b634e487b7160e01b600052603260045260246000fd5b60006020828403121561184d5761184d610ccc565b8151610dfc81610f35565b634e487b7160e01b600052601160045260246000fd5b60006001820161188057611880611858565b5060010190565b818103818111156105f9576105f9611858565b808201808211156105f9576105f9611858565b600060c08301825184526020808401516001600160401b0380821683880152604091508186015160018060a01b03808216848a015260609150828289015116828a0152608080890151818b015260a089015160c060a08c0152878151808a5260e08d0191508883019950600092505b8083101561195d5789518051881683528981015188168a8401528881015186168984015286015187168683015298880198600192909201919083019061191c565b509b9a5050505050505050505050565b600081518084526020808501945080840160005b838110156115fa5781516001600160801b03191687529582019590820190600101611981565b6060815260006119ba60608301866118ad565b6001600160401b038516602084015282810360408401526119db818561196d565b9695505050505050565b600082601f8301126119f9576119f9610f62565b8151611a0761105b826113cf565b818152846020838601011115611a1f57611a1f61137a565b61084382602083016020870161121d565b600060208284031215611a4557611a45610ccc565b81516001600160401b03811115611a5e57611a5e610d1c565b610843848285016119e5565b602081526000610dfc602083018461196d565b6001600160801b0319831681526040602082015260006108436040830184611241565b6001600160801b0319841681526001600160401b03831660208201526060604082015260006109c060608301846115c1565b6001600160e01b0319831681528151600090611af581600485016020870161121d565b919091016004019392505050565b60008251611b1581846020870161121d565b9190910192915050565b6001600160a01b038316815260406020820181905260009061084390830184611241565b600060608284031215611b5857611b58610e03565b611b60610e92565b90508151611b6d81611280565b8152602082810151611b7e81610f35565b8282015260408301516001600160401b03811115611b9e57611b9e610ee4565b8301601f81018513611bb257611bb2610f62565b8051611bc061105b82610fbb565b81815260059190911b82018301908381019087831115611be257611be2610fde565b928401925b82841015611c09578351611bfa81610f4d565b82529284019290840190611be7565b6040860152509295945050505050565b60006020808385031215611c2f57611c2f610ccc565b82516001600160401b0380821115611c4957611c49610d1c565b818501915085601f830112611c6057611c60610f62565b8151611c6e61105b82610fbb565b81815260059190911b83018401908481019088831115611c9057611c90610fde565b8585015b8381101561181557805185811115611cae57611cae610f62565b611cbc8b89838a0101611b43565b845250918601918601611c94565b6001600160401b038316815260406020820152600061084360408301846115c1565b600060208284031215611d0157611d01610ccc565b81516001600160401b03811115611d1a57611d1a610d1c565b61084384828501611b43565b6001600160801b031984168152606060208201526000611d496060830185611241565b82810360408401526119db8185611241565b604081526000611d6e60408301856118ad565b90506001600160801b0319831660208301529392505050565b60008060408385031215611d9d57611d9d610ccc565b82516001600160401b0380821115611db757611db7610d1c565b611dc3868387016119e5565b93506020850151915080821115611ddc57611ddc610d1c565b50611498858286016119e556fea2646970667358221220a2edd0ecfcf8eaa0cc141e129cd10cc0b7a1839d98d79f3747d7d4858f80b26c64736f6c63430008130033"
const MevShareBidContractCode string = "0x6080604052600436106100555760003560e01c8063274e3b041461005a5780633c34db651461008357806367d7d785146100a557806392f07a58146100c55780639b0a882e146100da578063e2a1d069146100fa575b600080fd5b61006d610068366004610e48565b61010d565b60405161007a9190610ee7565b60405180910390f35b34801561008f57600080fd5b506100a361009e366004610f12565b610375565b005b3480156100b157600080fd5b506100a36100c0366004610fbe565b6103c9565b3480156100d157600080fd5b5061006d610466565b3480156100e657600080fd5b506100a36100f5366004611045565b61048e565b61006d6101083660046110b4565b610528565b6060610117610863565b61012057600080fd5b6000306001600160a01b03166392f07a586040518163ffffffff1660e01b81526004016000604051808303816000875af1158015610162573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f1916820160405261018a9190810190611117565b9050600080610198836108c7565b91509150816101c25760405162461bcd60e51b81526004016101b99061118d565b60405180910390fd5b60006101cd8461099b565b9050600061020a8888604051806040016040528060168152602001756d657673686172653a76303a65746842756e646c657360501b815250610a6b565b90506102498160000151604051806040016040528060168152602001756d657673686172653a76303a65746842756e646c657360501b81525087610b50565b8051604080518082018252601f81527f6d657673686172653a76303a65746842756e646c6553696d526573756c74730060208083019190915282516001600160401b038816918101919091526102b09392015b604051602081830303815290604052610b50565b60008051602061163c8339815191528160000151826020015183604001516040516102dd93929190611213565b60405180910390a180516040517fdab8306bad2ca820d05b9eff8da2e3016d372c15f00bb032f758718b9cda395091610317918590611245565b60405180910390a1604051634d85441760e11b9061033b90839085906020016112a4565b60408051601f198184030181529082905261035992916020016112c9565b6040516020818303038152906040529550505050505092915050565b60008051602061163c83398151915261039160208301836112fa565b6103a16040840160208501611317565b6103ae6040850185611334565b6040516103be9493929190611384565b60405180910390a150565b60008051602061163c8339815191526103e560208501856112fa565b6103f56040860160208701611317565b6104026040870187611334565b6040516104129493929190611384565b60405180910390a17f417b0c16c40ca502ef10ae6921892668f006f527e3f4599cb95b9de965d4133461044860208501856112fa565b8383604051610459939291906113f9565b60405180910390a1505050565b60606000610472610c02565b9050808060200190518101906104889190611117565b91505090565b60008051602061163c8339815191526104aa60208401846112fa565b6104ba6040850160208601611317565b6104c76040860186611334565b6040516104d79493929190611384565b60405180910390a17fdab8306bad2ca820d05b9eff8da2e3016d372c15f00bb032f758718b9cda395061050d60208401846112fa565b8260405161051c929190611245565b60405180910390a15050565b6060610532610863565b61053b57600080fd5b6000306001600160a01b03166392f07a586040518163ffffffff1660e01b81526004016000604051808303816000875af115801561057d573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f191682016040526105a59190810190611117565b90506000806105b3836108c7565b91509150816105d45760405162461bcd60e51b81526004016101b99061118d565b60006105df8461099b565b9050600061061c8989604051806040016040528060168152602001756d657673686172653a76303a65746842756e646c657360501b815250610a6b565b905061065b8160000151604051806040016040528060168152602001756d657673686172653a76303a65746842756e646c657360501b81525087610b50565b8051604080518082018252601f81527f6d657673686172653a76303a65746842756e646c6553696d526573756c74730060208083019190915282516001600160401b038816918101919091526106b293920161029c565b60408051600280825260608201835260009260208301908036833701905050905087816000815181106106e7576106e761142e565b6001600160801b03199092166020928302919091019091015281518151829060019081106107175761071761142e565b60200260200101906001600160801b03191690816001600160801b0319168152505060006107748b8b604051806040016040528060168152602001756d657673686172653a76303a6d65726765644269647360501b815250610a6b565b90506107be8160000151604051806040016040528060168152602001756d657673686172653a76303a6d65726765644269647360501b8152508460405160200161029c9190611444565b60006107f88a604051806040016040528060168152602001756d657673686172653a76303a65746842756e646c657360501b815250610c63565b905060006108058261099b565b90506367d7d78560e01b85828860405160200161082493929190611492565b60408051601f198184030181529082905261084292916020016112c9565b60405160208183030381529060405299505050505050505050509392505050565b6040516000908190819063420100009082818181855afa9150503d80600081146108a9576040519150601f19603f3d011682016040523d82523d6000602084013e6108ae565b606091505b5091509150816108bd57600080fd5b6020015192915050565b60008060008063421000006001600160a01b0316856040516108e991906114cb565b600060405180830381855afa9150503d8060008114610924576040519150601f19603f3d011682016040523d82523d6000602084013e610929565b606091505b50915091508161097b5760405162461bcd60e51b815260206004820152601860248201527f42756e646c652073696d756c6174696f6e206661696c6564000000000000000060448201526064016101b9565b818180602001905181019061099091906114e7565b935093505050915091565b606060008063421000376001600160a01b0316846040516020016109bf9190610ee7565b60408051601f19818403018152908290526109d9916114cb565b600060405180830381855afa9150503d8060008114610a14576040519150601f19603f3d011682016040523d82523d6000602084013e610a19565b606091505b509150915081610a645760405162461bcd60e51b8152602060048201526016602482015275121a5b9d08195e1d1c9858dd1a5bdb8819985a5b195960521b60448201526064016101b9565b9392505050565b6040805160608082018352600080835260208301529181019190915260008063420300006001600160a01b0316868686604051602001610aad93929190611504565b60408051601f1981840301815290829052610ac7916114cb565b600060405180830381855afa9150503d8060008114610b02576040519150601f19603f3d011682016040523d82523d6000602084013e610b07565b606091505b509150915081610b32576342030000816040516375fff46760e01b81526004016101b9929190611526565b80806020019051810190610b46919061154a565b9695505050505050565b60008063420200006001600160a01b0316858585604051602001610b76939291906113f9565b60408051601f1981840301815290829052610b90916114cb565b600060405180830381855afa9150503d8060008114610bcb576040519150601f19603f3d011682016040523d82523d6000602084013e610bd0565b606091505b509150915081610bfb576342020000816040516375fff46760e01b81526004016101b9929190611526565b5050505050565b604051606090600090819063420100019082818181855afa9150503d8060008114610c49576040519150601f19603f3d011682016040523d82523d6000602084013e610c4e565b606091505b509150915081610c5d57600080fd5b92915050565b606060008063420200016001600160a01b03168585604051602001610c89929190611245565b60408051601f1981840301815290829052610ca3916114cb565b600060405180830381855afa9150503d8060008114610cde576040519150601f19603f3d011682016040523d82523d6000602084013e610ce3565b606091505b509150915081610d0e576342020001816040516375fff46760e01b81526004016101b9929190611526565b949350505050565b6001600160401b0381168114610d2b57600080fd5b50565b634e487b7160e01b600052604160045260246000fd5b604051606081016001600160401b0381118282101715610d6657610d66610d2e565b60405290565b604051601f8201601f191681016001600160401b0381118282101715610d9457610d94610d2e565b604052919050565b60006001600160401b03821115610db557610db5610d2e565b5060051b60200190565b6001600160a01b0381168114610d2b57600080fd5b600082601f830112610de557600080fd5b81356020610dfa610df583610d9c565b610d6c565b82815260059290921b84018101918181019086841115610e1957600080fd5b8286015b84811015610e3d578035610e3081610dbf565b8352918301918301610e1d565b509695505050505050565b60008060408385031215610e5b57600080fd5b8235610e6681610d16565b915060208301356001600160401b03811115610e8157600080fd5b610e8d85828601610dd4565b9150509250929050565b60005b83811015610eb2578181015183820152602001610e9a565b50506000910152565b60008151808452610ed3816020860160208601610e97565b601f01601f19169290920160200192915050565b602081526000610a646020830184610ebb565b600060608284031215610f0c57600080fd5b50919050565b600060208284031215610f2457600080fd5b81356001600160401b03811115610f3a57600080fd5b610d0e84828501610efa565b60006001600160401b03821115610f5f57610f5f610d2e565b50601f01601f191660200190565b600082601f830112610f7e57600080fd5b8135610f8c610df582610f46565b818152846020838601011115610fa157600080fd5b816020850160208301376000918101602001919091529392505050565b600080600060608486031215610fd357600080fd5b83356001600160401b0380821115610fea57600080fd5b610ff687838801610efa565b9450602086013591508082111561100c57600080fd5b61101887838801610f6d565b9350604086013591508082111561102e57600080fd5b5061103b86828701610f6d565b9150509250925092565b6000806040838503121561105857600080fd5b82356001600160401b038082111561106f57600080fd5b61107b86838701610efa565b9350602085013591508082111561109157600080fd5b50610e8d85828601610f6d565b6001600160801b031981168114610d2b57600080fd5b6000806000606084860312156110c957600080fd5b83356110d481610d16565b925060208401356001600160401b038111156110ef57600080fd5b6110fb86828701610dd4565b925050604084013561110c8161109e565b809150509250925092565b60006020828403121561112957600080fd5b81516001600160401b0381111561113f57600080fd5b8201601f8101841361115057600080fd5b805161115e610df582610f46565b81815285602083850101111561117357600080fd5b611184826020830160208601610e97565b95945050505050565b60208082526022908201527f62756e646c6520646f6573206e6f742073696d756c61746520636f72726563746040820152616c7960f01b606082015260800190565b600081518084526020808501945080840160005b838110156112085781516001600160a01b0316875295820195908201906001016111e3565b509495945050505050565b6001600160801b0319841681526001600160401b038316602082015260606040820152600061118460608301846111cf565b6001600160801b031983168152604060208201526000610d0e6040830184610ebb565b6001600160801b031981511682526001600160401b0360208201511660208301526000604082015160606040850152610d0e60608501826111cf565b6040815260006112b76040830185611268565b82810360208401526111848185610ebb565b6001600160e01b03198316815281516000906112ec816004850160208701610e97565b919091016004019392505050565b60006020828403121561130c57600080fd5b8135610a648161109e565b60006020828403121561132957600080fd5b8135610a6481610d16565b6000808335601e1984360301811261134b57600080fd5b8301803591506001600160401b0382111561136557600080fd5b6020019150600581901b360382131561137d57600080fd5b9250929050565b6000606082016001600160801b03198716835260206001600160401b03871681850152606060408501528185835260808501905086925060005b868110156113ec5783356113d181610dbf565b6001600160a01b0316825292820192908201906001016113be565b5098975050505050505050565b6001600160801b03198416815260606020820152600061141c6060830185610ebb565b8281036040840152610b468185610ebb565b634e487b7160e01b600052603260045260246000fd5b6020808252825182820181905260009190848201906040850190845b818110156114865783516001600160801b03191683529284019291840191600101611460565b50909695505050505050565b6060815260006114a56060830186611268565b82810360208401526114b78186610ebb565b90508281036040840152610b468185610ebb565b600082516114dd818460208701610e97565b9190910192915050565b6000602082840312156114f957600080fd5b8151610a6481610d16565b6001600160401b038416815260606020820152600061141c60608301856111cf565b6001600160a01b0383168152604060208201819052600090610d0e90830184610ebb565b6000602080838503121561155d57600080fd5b82516001600160401b038082111561157457600080fd5b908401906060828703121561158857600080fd5b611590610d44565b825161159b8161109e565b8152828401516115aa81610d16565b818501526040830151828111156115c057600080fd5b80840193505086601f8401126115d557600080fd5b825191506115e5610df583610d9c565b82815260059290921b8301840191848101908884111561160457600080fd5b938501935b8385101561162b57845161161c81610dbf565b82529385019390850190611609565b604083015250969550505050505056fe83481d5b04dea534715acad673a8177a46fc93882760f36bdc16ccac439d504ea264697066735822122010590eb1a24c5c4a46d40f808f203d27bffdf7a28c48203bc19ac28b8c6611c664736f6c63430008140033"
