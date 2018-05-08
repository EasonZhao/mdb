package main

import (
	"github.com/astaxie/beego/orm"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
)

const (
	COINBASE = iota
	TRANSACTION
)

const (
	EMPTY_HASH = "0000000000000000000000000000000000000000000000000000000000000000"
)

type Block struct {
	Id           int32
	Hash         string `orm:"unique;size(64)"`
	PrevBlock    string `orm:"size(64)"`
	Transactions []*Tx  `orm:"reverse(many)"`
	Height       int    `orm:"default(-1)"`
}

type Tx struct {
	Id          int32
	Hash        string   `orm:"unique;size(64)"`
	TxIn        []*TxIn  `orm:"reverse(many)"`
	TxOut       []*TxOut `orm:"reverse(many)"`
	Block       *Block   `orm:"rel(fk)"`
	HashWitness bool     `orm:"default(false)"`
	Size        int      `orm:"default(0)"`
	IsCoinBase  bool     `orm:"default(false)"`
	Version     int32
	LockTime    uint32
}

type TxIn struct {
	Id          int32
	Transaction *Tx    `orm:"rel(fk)"`
	Hash        string `orm:"size(64)"`
	Index       int    `orm:"default(-1)"`
	Address     string `orm:"size(40)"`
	Value       int64
	Sequence    uint32
}

type TxOut struct {
	Id          int32
	Transaction *Tx    `orm:"rel(fk)"`
	Address     string `orm:"size(40)"`
	Value       int64
}

func newTransaction(tx *btcutil.Tx) (*Tx, error) {
	result := new(Tx)
	msgTx := tx.MsgTx()

	result.Hash = msgTx.TxHash().String()
	result.Version = msgTx.Version
	result.LockTime = msgTx.LockTime
	result.HashWitness = msgTx.HasWitness()
	result.Size = msgTx.SerializeSize()
	result.IsCoinBase = blockchain.IsCoinBaseTx(msgTx)

	result.TxIn = make([]*TxIn, len(msgTx.TxIn))
	for i, in := range msgTx.TxIn {
		iv := new(TxIn)
		iv.Transaction = result
		iv.Hash = in.PreviousOutPoint.Hash.String()
		iv.Index = int(in.PreviousOutPoint.Index)
		iv.Sequence = in.Sequence
		result.TxIn[i] = iv
	}
	result.TxOut = make([]*TxOut, len(msgTx.TxOut))
	for i, out := range msgTx.TxOut {
		ov := new(TxOut)
		ov.Transaction = result
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(out.PkScript, &chaincfg.TestNet3Params)
		if err != nil {
			panic(err)
		}
		ov.Value = out.Value
		if len(addrs) > 0 {
			ov.Address = addrs[0].EncodeAddress()
		}
		result.TxOut[i] = ov
	}
	return result, nil
}

func NewBlock(b *btcutil.Block) (*Block, error) {
	result := Block{
		Hash:      b.Hash().String(),
		PrevBlock: b.MsgBlock().Header.PrevBlock.String(),
		Height:    -1,
	}
	result.Transactions = make([]*Tx, len(b.Transactions()))
	for i, tx := range b.Transactions() {
		t, _ := newTransaction(tx)
		t.Block = &result
		result.Transactions[i] = t
	}
	return &result, nil
}

func init() {
	orm.RegisterModel(new(Block), new(Tx), new(TxIn), new(TxOut))
}
