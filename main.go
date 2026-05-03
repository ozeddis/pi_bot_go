package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "time"

    "github.com/stellar/go/clients/horizonclient"
    "github.com/stellar/go/keypair"
    "github.com/stellar/go/txnbuild"
)

type PiNetworkSDK struct {
    BaseURL           string
    NetworkPassphrase string
    HorizonURL        string
    Headers           map[string]string
    Client            *http.Client
}

func NewStandaloneSDK() *PiNetworkSDK {
    return &PiNetworkSDK{
        BaseURL:           "https://api.mainnet.minepi.com",
        NetworkPassphrase: "Pi Network",
        HorizonURL:        "https://api.mainnet.minepi.com",
        Headers:           map[string]string{},
        Client: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (sdk *PiNetworkSDK) SubmitTransaction(txXDR string) error {
    requestBody := map[string]string{
        "tx": txXDR,
    }
    jsonData, _ := json.Marshal(requestBody)

    req, err := http.NewRequest("POST", sdk.HorizonURL+"/transactions", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    for k, v := range sdk.Headers {
        req.Header.Set(k, v)
    }

    resp, err := sdk.Client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    if resp.StatusCode >= 400 {
        return fmt.Errorf("❌ HTTP %d: %s", resp.StatusCode, body)
    }

    fmt.Println("✅ Submission response:", string(body))
    return nil
}

func main() {
    const destination = "MDFNWH6ZFJVHJDLBMNOUT35X4EEKQVJAO3ZDL4NL7VQJLC4PJOQFWAAAAAANJO4A74EMO"
    const amount = "0.7437131"
    const secret = ""

    sdk := NewStandaloneSDK()
    kp := keypair.MustParseFull(secret)

    client := horizonclient.Client{HorizonURL: sdk.HorizonURL}
    account, err := client.AccountDetail(horizonclient.AccountRequest{AccountID: kp.Address()})
    if err != nil {
        log.Fatal("❌ Failed to load account:", err)
    }

    paymentOp := txnbuild.Payment{
        Destination: destination,
        Amount:      amount,
        Asset:       txnbuild.NativeAsset{},
    }

    feeStats, err := client.FeeStats()
    if err != nil {
        log.Fatal("❌ Failed to fetch fee stats:", err)
    }
    // baseFee := txnbuild.BaseFee(feeStats.FeeCharged.P90)

    tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
        SourceAccount:        &account,
        Operations:           []txnbuild.Operation{&paymentOp},
        BaseFee: feeStats.FeeCharged.P90,

        Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(30)},
        IncrementSequenceNum: true,
    })
    if err != nil {
        log.Fatal("❌ Transaction creation failed:", err)
    }

    tx, err = tx.Sign(sdk.NetworkPassphrase, kp)
    if err != nil {
        log.Fatal("❌ Signing failed:", err)
    }

    txXDR, err := tx.Base64()
    if err != nil {
        log.Fatal("❌ Could not encode transaction to XDR:", err)
    }

    if err := sdk.SubmitTransaction(txXDR); err != nil {
        log.Println(err)
    }
}
