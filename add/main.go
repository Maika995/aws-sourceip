package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

// IAMポリシーの構造体
type Policy struct {
	Version string
	Statement []Statement
}

type Statement struct {
	Sid string
	Effect string
	Action string
	Resource string
	Condition Condition
}

type Condition struct {
	NotIpAddress NotIpAddress
}

type NotIpAddress struct {
	SourceIp []string `json:"aws:SourceIp"`
}

type MyEvent struct {
	SourceIp string
}

func ListPolicyVersion(svc iamiface.IAMAPI)(latest, oldest string) {
	// os.Getenv（）でLambdaの環境変数を取得
	arn := os.Getenv("arn") // IP制限IAMポリシーarn
	input := iam.ListPolicyVersionsInput{
		PolicyArn: aws.String(arn),
	}
	result, err := svc.ListPolicyVersions(&input)
	if err != nil {
		log.Println(err.Error())
	}
	count := len(result.Versions)

	return *result.Versions[0].VersionId, *result.Versions[count-1].VersionId
}

func GetPolicy(svc iamiface.IAMAPI, latest string)(latestPolicy Policy){
	arn := os.Getenv("arn")
	versionID := latest
	input := iam.GetPolicyVersionInput{
		PolicyArn: aws.String(arn),
		VersionId: aws.String(versionID),
	}

	result, err := svc.GetPolicyVersion(&input)
	if err != nil {
		log.Println(err.Error())
	}

	Document, err := url.QueryUnescape(*result.PolicyVersion.Document)
	bytes := []byte(Document)
	var policy Policy
	json.Unmarshal(bytes, &policy)

	return policy
}

func DeletePolicyVersion(svc iamiface.IAMAPI, oldest string)  {
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

func CreatePolicy(svc iamiface.IAMAPI, policy Policy, event MyEvent)  {
	arn := os.Getenv("arn")

	policy.Statement[0].Condition.NotIpAddress.SourceIp = append(policy.Statement[0].Condition.NotIpAddress.SourceIp, event.SourceIp)
	JSONdata, _ := json.Marshal(policy)
	newPolicy := string(JSONdata)

	setAsDefault := true

	input := iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String(arn),
		PolicyDocument: aws.String(newPolicy),
		SetAsDefault: aws.Bool(setAsDefault),
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
func run(event MyEvent) (interface{}, error) {
	log.Println("--- 開始")
	log.Println("----- セッション作成")
	svc := iam.New(session.Must(session.NewSession()))

	log.Println("----- ポリシーバージョン取得")
	latest, oldest := ListPolicyVersion(svc)

	log.Println("----- 現在の最新のポリシー取得")
	policy := GetPolicy(svc, latest)

	log.Println("----- 古いポリシーのバージョンを削除")
	DeletePolicyVersion(svc, oldest)

	log.Println("----- IPアドレスを追加したポリシーに書き換え")
	CreatePolicy(svc, policy, event)

	response := "IPアドレスの追加が完了しました！"

	log.Println("--- 完了")
	return response, nil
}

/**************************
    メイン
**************************/
func main() {
	lambda.Start(run)
}