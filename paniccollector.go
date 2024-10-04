package paniccollector

import (
	"bytes"
	"context"
	"crypto"
	"fmt"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	pb "github.com/brandcompete/panic-collector/paniccollector"
	"github.com/bugsnag/panicwrap"
	"google.golang.org/grpc"
	"io"
	"os"
	"time"
)

type Config struct {
	GrpcServerAddr string
}

var panicCollectorConfig *Config
var grpcClient pb.PanicCollectorClient

func Initialize(config *Config) error {
	panicCollectorConfig = config

	client, err := grpc.NewClient(panicCollectorConfig.GrpcServerAddr, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %v", err)
	}

	grpcClient = pb.NewPanicCollectorClient(client)

	exitStatus, err := panicwrap.BasicWrap(panicHandler)
	if err != nil {
		return fmt.Errorf("error setting up panic wrap: %v", err)
	}

	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}

	return nil
}

func panicHandler(output string) {
	publicKey, err := fetchPublicKeyFromGrpc()
	if err != nil {
		fmt.Printf("Failed to fetch public key: %v\n", err)
		os.Exit(1)
	}

	encryptedOutput, err := encryptWithPGP(output, publicKey)
	if err != nil {
		fmt.Printf("Failed to encrypt panic output: %v\n", err)
		os.Exit(1)
	}

	err = sendToGrpcBackend(encryptedOutput)
	if err != nil {
		fmt.Printf("Failed to send panic data to backend: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("The application panicked, encrypted data sent to backend.\n")
	os.Exit(1)
}

func fetchPublicKeyFromGrpc() (string, error) {
	req := &pb.PublicKeyRequest{}
	resp, err := grpcClient.GetPublicKey(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch public key from gRPC server: %v", err)
	}
	return resp.PublicKey, nil
}

func encryptWithPGP(plaintext string, publicKeyArmor string) (string, error) {
	pubKeyReader := bytes.NewBufferString(publicKeyArmor)
	entityList, err := openpgp.ReadArmoredKeyRing(pubKeyReader)
	if err != nil {
		return "", fmt.Errorf("error reading public key: %v", err)
	}

	var encryptedBuf bytes.Buffer

	armorWriter, err := armor.Encode(&encryptedBuf, "PGP MESSAGE", nil)
	if err != nil {
		return "", fmt.Errorf("error creating armor: %v", err)
	}

	config := &packet.Config{
		DefaultCipher: packet.CipherAES256,
		DefaultHash:   crypto.SHA256,
		Time:          time.Now,
	}

	plaintextWriter, err := openpgp.Encrypt(armorWriter, entityList, nil, nil, config)
	if err != nil {
		return "", fmt.Errorf("error creating encryption writer: %v", err)
	}

	_, err = io.Copy(plaintextWriter, bytes.NewBufferString(plaintext))
	if err != nil {
		return "", fmt.Errorf("error encrypting data: %v", err)
	}

	err = plaintextWriter.Close()
	if err != nil {
		return "", fmt.Errorf("error closing encryption writer: %v", err)
	}

	err = armorWriter.Close()
	if err != nil {
		return "", fmt.Errorf("error closing armor writer: %v", err)
	}

	return encryptedBuf.String(), nil
}

func sendToGrpcBackend(encryptedData string) error {
	req := &pb.PanicRequest{
		EncryptedData: encryptedData,
	}

	_, err := grpcClient.CollectPanic(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to send panic to gRPC server: %v", err)
	}

	return nil
}
