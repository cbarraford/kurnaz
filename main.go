package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/piotrnar/gocoin/lib/btc"
)

var WORDS []string

func init() {
	rand.Seed(time.Now().UnixNano())
}

type AddressInfo struct {
	Word          string
	Hash160       string
	Address       string
	Key           string // Hex encoded
	NTx           int    `json:"n_tx"`
	TotalReceived int    `json:"total_received"`
	TotalSent     int    `json:"total_send"`
	FinalBalance  int    `json:"final_balance"`
}

type Response struct {
	AddressInfos []AddressInfo `json:"addresses"`
}

func main() {
	fileName := flag.String("file", "words.txt", "Path of the file containing words")
	frequency := flag.String("frequency", "1333ms", "Check sleep duration")
	flag.Parse()

	file, err := os.Open(*fileName)
	if err != nil {
		panic(err)
	}

	pid := strconv.Itoa(os.Getpid())
	err = os.Mkdir(pid, os.ModePerm)
	if err != nil {
		panic(err)
	}

	configFile, err := os.Create(pid + "/conf")
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(configFile, "Frequency: "+*frequency)

	allRestulsFile, err := os.Create(pid + "/all.csv")
	if err != nil {
		panic(err)
	}

	usedRestulsFile, err := os.Create(pid + "/used.csv")
	if err != nil {
		panic(err)
	}

	activeRestulsFile, err := os.Create(pid + "/active.csv")
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		WORDS = append(WORDS, scanner.Text())
	}

	sleepDuration, err := time.ParseDuration(*frequency)
	if err != nil {
		panic(err)
	}

	for {

		seed := getSeed(12)

		addressInfo := newAddressInfoFromWord(seed)

		recordAddressInfo(allRestulsFile, addressInfo)

		if addressInfo.TotalReceived > 0 {
			fmt.Printf("Found Used Wallet\n")
			recordAddressInfo(usedRestulsFile, addressInfo)
		}

		if addressInfo.FinalBalance > 0 {
			fmt.Printf("Found Active Wallet!!!\n")
			recordAddressInfo(activeRestulsFile, addressInfo)
		}

		time.Sleep(sleepDuration)
	}

}

func recordAddressInfo(writer io.Writer, addressInfo *AddressInfo) {
	fmt.Fprintf(
		writer,
		"%s, %s, %s, %f, %f\n",
		addressInfo.Key,
		addressInfo.Address,
		addressInfo.Word,
		float64(addressInfo.TotalReceived)/100000000,
		float64(addressInfo.FinalBalance)/100000000,
	)
}

// Using Blockexplorer's api
func generateHashFromPublicKey(publicKey []byte) string {
	publicKeyString := hex.EncodeToString(publicKey)
	//resp, err := http.Get("http://blockchain.info/q/hashpubkey/" + publicKeyString)
	resp, err := http.Get("http://blockexplorer.com/q/hashpubkey/" + publicKeyString)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	hash, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(hash)
}

func newAddressInfoFromWord(word string) *AddressInfo {
	sha256Hash := sha256.New()
	_, err := sha256Hash.Write([]byte(word))
	if err != nil {
		panic(err)
	}
	privateKey := sha256Hash.Sum(nil)

	publicKey := btc.PublicFromPrivate(privateKey, false)

	address := btc.NewAddrFromPubkey(publicKey, 0x00).String()

	resp, err := http.Get("http://blockchain.info/address/" + address + "?format=json")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var addressInfo AddressInfo
	err = json.Unmarshal(body, &addressInfo)
	if err != nil {
		return &addressInfo
	}

	addressInfo.Word = word
	addressInfo.Key = hex.EncodeToString(privateKey)

	return &addressInfo
}

func getSeed(count int) string {
	var words []string
	for i := 1; i <= count; i++ {
		index := rand.Intn(len(WORDS))
		words = append(words, WORDS[index])
	}
	return strings.Join(words, " ")
}
