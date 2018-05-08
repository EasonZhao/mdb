package main

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
)

const (
	DBUrl   = "uc_admin:admin123@/usercenter?charset=utf8"
	ADDRESS = "n3KZTn5bfu37DuZ1jPJMbaqeeUgL951JNm"
)

func init() {
	err := orm.RegisterDriver("mysql", orm.DRMySQL)
	if err != nil {

		panic(err)
	}

	err = orm.RegisterDataBase("default", "mysql", DBUrl)
	if err != nil {
		panic(err)
	}
	//orm.Debug = true
}

func main() {
	o := orm.NewOrm()
	var tx_ins []*TxIn
	_, err := o.QueryTable("tx_in").Filter("address", ADDRESS).All(&tx_ins)
	if err != nil {
		panic(err)
	}
	fmt.Println("tx:")
	for i, _ := range tx_ins {
		o.LoadRelated(tx_ins[i], "Transaction")
		fmt.Println(tx_ins[i].Transaction.Hash)
	}

	var tx_outs []*TxOut
	_, err = o.QueryTable("tx_out").Filter("address", ADDRESS).All(&tx_outs)
	if err != nil {
		panic(err)
	}
	for i, _ := range tx_outs {
		o.LoadRelated(tx_outs[i], "Transaction")
		fmt.Println(tx_outs[i].Transaction.Hash)
	}
}
