package main

import (
	"encoding/json"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
	_ "github.com/go-sql-driver/mysql"
	zmq "github.com/pebbe/zmq4"
	"os"
)

const (
	DBUrl      = "uc_admin:admin123@/usercenter?charset=utf8"
	CACHE_SIZE = 5000000
)

var cache map[string]int
var o orm.Ormer

type Configuration struct {
	DBPath string `json:"DB_PATH"`
}

func init() {
	//init log
	logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/debug.log","level":7,"maxlines":0,"maxsize":0,"daily":true,"maxdays":10}`)
	logs.Async(1e3)

	config := loadConf("./conf.json")

	//init orm
	err := orm.RegisterDriver("mysql", orm.DRMySQL)
	if err != nil {
		logs.Critical(err)
		panic(err)
	}

	err = orm.RegisterDataBase("default", "mysql", config.DBPath)
	if err != nil {
		logs.Critical(err)
		panic(err)
	}

	cache = make(map[string]int, CACHE_SIZE)
}

func loadConf(path string) *Configuration {
	file, _ := os.Open(path)
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := Configuration{}
	err := decoder.Decode(&config)
	if err != nil {
		logs.Critical("load configure err = ", err)
		panic(err)
	}
	return &config
}

func calcInput(b *Block) bool {
	if b.PrevBlock == EMPTY_HASH {
		b.Height = 0
		cache[b.Hash] = b.Height
		return true
	}
	b.Height = cache[b.PrevBlock] + 1
	cache[b.Hash] = b.Height

	for _, tx := range b.Transactions {
		for i, _ := range tx.TxIn {
			iv := tx.TxIn[i]
			if iv.Hash != EMPTY_HASH {
				t := Tx{Hash: iv.Hash}
				err := o.Read(&t, "hash")
				if err == orm.ErrNoRows {
					return false
				}
				_, err = o.LoadRelated(&t, "Txout")
				if err != nil {
					logs.Error("LoadRelated failue hash = ", t.Hash, " err = ", err)
				}
				iv.Address = t.TxOut[iv.Index].Address
				iv.Value = t.TxOut[iv.Index].Value
			}
		}
	}
	return true
}

func writeToDB(b *Block) bool {
	o.Begin()
	o.Insert(b)
	for _, t := range b.Transactions {
		o.Insert(t)
		for _, iv := range t.TxIn {
			o.Insert(iv)
		}
		for _, ov := range t.TxOut {
			o.Insert(ov)
		}
	}
	err := o.Commit()
	if err != nil {
		logs.Error("Insert mysql failure, err = ", err)
		return false
	}
	return true
}

func zmq_receive(c *chan *Block) {
	subscriber, _ := zmq.NewSocket(zmq.SUB)
	defer subscriber.Close()
	subscriber.Connect("tcp://127.0.0.1:28332")
	subscriber.SetSubscribe("rawblock")
	for {
		msgs, e := subscriber.RecvMessageBytes(0)
		if e != nil {
			logs.Critical("RecvMessageBytes failure, err = ", e)
			return
		}
		cmd := string(msgs[0])

		if cmd == "END" {
			break
		} else if cmd == "rawblock" {
			wb, err := btcutil.NewBlockFromBytes(msgs[1])
			if err != nil {
				logs.Error("NewBlockFromBytes failure, hash = ", base58.Encode(msgs[1]), ", err = ", err)
				continue
			}
			block, err := NewBlock(wb)
			if err != nil {
				logs.Error("NewBlock failure, hash = ", wb.Hash().String(), ", err = ", err)
				continue
			}
			*c <- block
		}
	}
}

func writeMysql(c *chan *Block) {
	for {
		block := <-*c
		calcInput(block)
		writeToDB(block)
		logs.Info("Hash: ", block.Hash, " height: ", block.Height)
	}
}

func main() {
	orm.RunCommand()
	o = orm.NewOrm()
	o.Using("dafalut")

	ch1 := make(chan *Block)
	go zmq_receive(&ch1)

	ch2 := make(chan *Block, 100000)
	go writeMysql(&ch2)
	for {
		select {
		case block := <-ch1:
			ch2 <- block
		}
	}
}
