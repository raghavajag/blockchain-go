This project implements a basic blockchain and cryptocurrency from scratch. The goal is to demonstrate a blockchain's core concepts and features by building a simplified version from the ground up.

![blockchain-arch](https://github.com/user-attachments/assets/da186159-8a48-4dda-825f-83a5c758f642)

## Features

- **Blockchain Data Structure**: Implements a linked list of blocks, where each block contains a hash pointer to the previous block, forming an immutable chain. Blocks store transaction data, timestamps, and nonce values.

- **Proof-of-Work Consensus**: Includes a proof-of-work algorithm for mining new blocks. Miners compete to find a nonce value that produces a block hash meeting a specified difficulty target. This process secures the blockchain by requiring computational work to add new blocks.

- **Transaction Management**: Supports creating, signing, and validating transactions. Transactions transfer value between addresses and are grouped into blocks. Includes transaction inputs, outputs, and digital signatures for authentication.

- **Wallet Functionality**: Provides a wallet system for managing key pairs and addresses. Wallets can create new key pairs, sign transactions, and check balances associated with their addresses on the blockchain.

- **Decentralized Network**: Implements a decentralized peer-to-peer network where nodes can join, share blockchain data, and synchronize their local copies of the blockchain. Nodes communicate via a simple protocol to relay transactions and blocks.

- **Consensus Validation**: Nodes validate incoming blocks and transactions according to consensus rules. Invalid blocks or transactions are rejected to maintain the integrity of the blockchain.

- **Persistence**: The blockchain data is persisted to disk using a simple database solution (e.g., BoltDB). This allows nodes to restore the blockchain state upon restart.

- **API Interface**: Exposes a RESTful API for interacting with the blockchain. Allows clients to submit transactions, query block and transaction data, and monitor the network status.


Data Flow Diagram
![image](https://github.com/user-attachments/assets/e7ac754c-f36c-4fa2-ad8d-dcca9648cea1)
