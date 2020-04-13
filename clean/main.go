package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// IAMポリシーの構造体
type Policy struct {
	Version   string
	Statement []Statement
}

type Statement struct {
	Sid       string
	Effect    string
	Action    string
	Resource  string
	Condition Condition
}

type Condition struct {
	NotIpAddress NotIpAddress
}

type NotIpAddress struct {
	SourceIp []string `json:"aws:SourceIp"`
}

func ListPolicyVersion(svc iamiface.IAMAPI) (oldest string) {
	arn := os.Getenv("arn") // IP制限IAMポリシーarn
	input := iam.ListPolicyVersionsInput{
		PolicyArn: aws.String(arn),
	}
	result, err := svc.ListPolicyVersions(&input)
	if err != nil {
		log.Println(err.Error())
	}
	count := len(result.Versions)

	return *result.Versions[count-1].VersionId
}

func DeletePolicyVersion(svc iamiface.IAMAPI, oldest string) {
	arn := os.Getenv("arn")
	versionID := oldest
	input := iam.DeletePolicyVersionInput{
		PolicyArn: aws.String(arn),
		VersionId: aws.String(versionID),
	}

	result, err := svc.DeletePolicyVersion(&input)
	if err != nil {
		log.Println(err.Error())
	}

	fmt.Println(result)
}

func GetResetPolicy(svc s3iface.S3API) (defaultPolicy Policy) {
	bucket := os.Getenv("bucket") // バケット名
	key := os.Getenv("key")       // キー

	input := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := svc.GetObject(&input)
	if err != nil {
		log.Println(err.Error())
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(result.Body)
	bytes := []byte(buf.String())
	var policy Policy
	json.Unmarshal(bytes, &policy)

	return policy
}

func PutPolicy(svc iamiface.IAMAPI, policy Policy) {
	arn := os.Getenv("arn")

	JSONdata, _ := json.Marshal(policy)
	newPolicy := string(JSONdata)

	setAsDefault := true

	input := iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String(arn),
		PolicyDocument: aws.String(newPolicy),
		SetAsDefault:   aws.Bool(setAsDefault),
	}

	result, err := svc.CreatePolicyVersion(&input)
	if err != nil {
		log.Println(err.Error())
	}

	fmt.Println(result)
}

/**************************
    処理実行
**************************/
func run() error {
	log.Println("--- 開始")
	log.Println("----- セッション作成")
	svcIam := iam.New(session.Must(session.NewSession()))
	svcS3 := s3.New(session.Must(session.NewSession()))

	log.Println("----- ポリシーバージョン取得")
	oldest := ListPolicyVersion(svcIam)

	log.Println("----- 古いポリシーのバージョンを削除")
	DeletePolicyVersion(svcIam, oldest)

	log.Println("----- デフォルトのポリシーをS3から取得")
	policy := GetResetPolicy(svcS3)

	log.Println("----- デフォルトのポリシーに書き換え")
	PutPolicy(svcIam, policy)

	log.Println("--- 完了")
	return nil
}

/**************************
    メイン
**************************/
func main() {
	lambda.Start(run)
}
