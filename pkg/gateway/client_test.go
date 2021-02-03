package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/kong/go-kong/kong"
)

var kongURL = "http://localhost:8001"
var Token = "xxxx"

func Test_createConsumer(t *testing.T) {
	clientBase := &http.Client{}
	defaultTransport := http.DefaultTransport.(*http.Transport)
	clientBase.Transport = defaultTransport
	headers := make(http.Header)
	headers.Set("Kong-Admin-Token", Token)
	client := kong.HTTPClientWithHeaders(clientBase, headers)
	kongClient, err := kong.NewClient(&kongURL, &client)

	if err != nil {
		print(err)
	}
	Username := "test"
	CustomID := "custom_id_test"
	consumer := &kong.Consumer{
		Username: &Username,
		CustomID: &CustomID,
	}
	ctx := context.Background()
	println("executing test")
	result, err := kongClient.Consumers.Create(ctx, consumer)
	println("executing test complete")
	fmt.Println(err)
	fmt.Println(result)
}

func Test_createConsumerKey(t *testing.T) {
	clientBase := &http.Client{}
	defaultTransport := http.DefaultTransport.(*http.Transport)
	clientBase.Transport = defaultTransport
	headers := make(http.Header)
	headers.Set("Kong-Admin-Token", Token)
	client := kong.HTTPClientWithHeaders(clientBase, headers)
	kongClient, err := kong.NewClient(&kongURL, &client)

	if err != nil {
		print(err)
	}

	id := "52b228d8-07c5-47f4-95a8-48572b1109fb"
	consumer := &kong.Consumer{

		ID: &id,
	}
	ctx := context.Background()
	println("executing test")
	result, err := kongClient.KeyAuths.Create(ctx, consumer.ID, &kong.KeyAuth{})
	println("Creating keyauth complete")
	fmt.Println(err)
	fmt.Println(result)
}

func Test_createConsumberGroup(t *testing.T) {
	clientBase := &http.Client{}
	defaultTransport := http.DefaultTransport.(*http.Transport)
	clientBase.Transport = defaultTransport
	headers := make(http.Header)
	headers.Set("Kong-Admin-Token", Token)
	client := kong.HTTPClientWithHeaders(clientBase, headers)

	kongClient, err := kong.NewClient(&kongURL, &client)

	if err != nil {
		print(err)
	}
	ctx := context.Background()
	groupName := "group3"
	acl := &kong.ACLGroup{
		Group: &groupName,
	}
	// acl, err = kongClient.ACLs.Create(ctx, &groupName, acl)
	// println("ACL creation complete")

	fmt.Println(err)
	fmt.Println(acl)
	Username := "test4"
	CustomID := "custom_id_test4"
	consumer := &kong.Consumer{
		Username: &Username,
		CustomID: &CustomID,
	}

	result, err := kongClient.Consumers.Create(ctx, consumer)

	fmt.Println(err)
	fmt.Println(result)
	println("Consumer creation complete")

	createdACL, err := kongClient.ACLs.Create(ctx, result.ID, acl)

	fmt.Println(err)
	fmt.Println(createdACL)
	println("Added ACL to consumber")

	keyauth, err := kongClient.KeyAuths.Create(ctx, result.ID, &kong.KeyAuth{})

	fmt.Println(err)
	fmt.Println(keyauth)
	println("Consumer key creation complete")

}

func Test_applyAclOnRoute(t *testing.T) {
	clientBase := &http.Client{}
	defaultTransport := http.DefaultTransport.(*http.Transport)
	clientBase.Transport = defaultTransport
	headers := make(http.Header)
	headers.Set("Kong-Admin-Token", Token)
	//client := kong.HTTPClientWithHeaders(clientBase, headers)

	//kongClient, err := kong.NewClient(&kongURL, &client)

	routeName := "apikeyauth"

	urlObj, _ := url.Parse(kongURL + "/routes" + "/" + routeName + "/plugins")
	fmt.Println(urlObj)

	req := &http.Request{
		Method: "GET",
		URL:    urlObj,
		Header: headers,
	}

	res, err := http.DefaultClient.Do(req)
	fmt.Println(err)
	body, err := ioutil.ReadAll(res.Body)
	//fmt.Println(res.Body)

	// data, next, err := kongClient.cli.list(ctx,
	// 	"/services/"+*serviceNameOrID+"/routes", opt)

	type list struct {
		Data []json.RawMessage `json:"data"`
	}

	plugins := list{}

	json.Unmarshal(body, &plugins)

	for _, object := range plugins.Data {
		b, err := object.MarshalJSON()
		fmt.Println(err)

		var plugin kong.Plugin
		err = json.Unmarshal(b, &plugin)

		fmt.Println(*plugin.ID)

		pluginName := *plugin.Name
		if pluginName == "acl" {
			fmt.Println("acl plugin")
			client := kong.HTTPClientWithHeaders(clientBase, headers)
			kongClient, err := kong.NewClient(&kongURL, &client)
			fmt.Println(err)
			ctx := context.Background()
			//allow := plugin.Config["allow"] + "grup4"

			val := plugin.Config["allow"]
			if v, ok := val.([]interface{}); ok {

				val = append(v, "group4")

			}

			fmt.Printf("%T\n", val)
			fmt.Println(plugin.Config["allow"])
			plugin.Config["allow"] = val
			fmt.Println(plugin.Config["allow"])
			updateRes, err := kongClient.Plugins.Create(ctx, &plugin)
			fmt.Println(updateRes)
			fmt.Println("update consumer access complete")

		}
	}

}
