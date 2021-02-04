package subscription_test

import (
	"testing"
	"time"

	"github.com/Axway/agents-kong/pkg/subscription/apikey"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-kong/pkg/subscription"
	"github.com/sirupsen/logrus"
)

var swagger = `
{
  "swagger": "2.0",
  "info": {
    "version": "1.0.0",
    "title": "Swagger Petstore",
    "description": "A sample API that uses a petstore as an example to demonstrate features in the swagger-2.0 specification",
    "termsOfService": "http://swagger.io/terms/",
    "contact": {
      "name": "Swagger API Team"
    },
    "license": {
      "name": "MIT"
    }
  },
  "host": "petstore.swagger.io",
  "basePath": "/api",
  "schemes": [
    "http"
  ],
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "paths": {
    "/pets": {
      "get": {
        "description": "Returns all pets from the system that the user has access to",
        "operationId": "findPets",
        "produces": [
          "application/json",
          "application/xml",
          "text/xml",
          "text/html"
        ],
        "parameters": [
          {
            "name": "tags",
            "in": "query",
            "description": "tags to filter by",
            "required": false,
            "type": "array",
            "items": {
              "type": "string"
            },
            "collectionFormat": "csv"
          },
          {
            "name": "limit",
            "in": "query",
            "description": "maximum number of results to return",
            "required": false,
            "type": "integer",
            "format": "int32"
          }
        ],
        "responses": {
          "200": {
            "description": "pet response",
            "schema": {
              "type": "array",
              "items": {
                "$ref": "#/definitions/Pet"
              }
            }
          },
          "default": {
            "description": "unexpected error",
            "schema": {
              "$ref": "#/definitions/ErrorModel"
            }
          }
        }
      },
      "post": {
        "description": "Creates a new pet in the store.  Duplicates are allowed",
        "operationId": "addPet",
        "produces": [
          "application/json"
        ],
        "parameters": [
          {
            "name": "pet",
            "in": "body",
            "description": "Pet to add to the store",
            "required": true,
            "schema": {
              "$ref": "#/definitions/NewPet"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "pet response",
            "schema": {
              "$ref": "#/definitions/Pet"
            }
          },
          "default": {
            "description": "unexpected error",
            "schema": {
              "$ref": "#/definitions/ErrorModel"
            }
          }
        }
      }
    },
    "/pets/{id}": {
      "get": {
        "description": "Returns a user based on a single ID, if the user does not have access to the pet",
        "operationId": "findPetById",
        "produces": [
          "application/json",
          "application/xml",
          "text/xml",
          "text/html"
        ],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "description": "ID of pet to fetch",
            "required": true,
            "type": "integer",
            "format": "int64"
          }
        ],
        "responses": {
          "200": {
            "description": "pet response",
            "schema": {
              "$ref": "#/definitions/Pet"
            }
          },
          "default": {
            "description": "unexpected error",
            "schema": {
              "$ref": "#/definitions/ErrorModel"
            }
          }
        }
      },
      "delete": {
        "description": "deletes a single pet based on the ID supplied",
        "operationId": "deletePet",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "description": "ID of pet to delete",
            "required": true,
            "type": "integer",
            "format": "int64"
          }
        ],
        "responses": {
          "204": {
            "description": "pet deleted"
          },
          "default": {
            "description": "unexpected error",
            "schema": {
              "$ref": "#/definitions/ErrorModel"
            }
          }
        }
      }
    }
  },
  "definitions": {
    "Pet": {
      "type": "object",
      "allOf": [
        {
          "$ref": "#/definitions/NewPet"
        },
        {
          "required": [
            "id"
          ],
          "properties": {
            "id": {
              "type": "integer",
              "format": "int64"
            }
          }
        }
      ]
    },
    "NewPet": {
      "type": "object",
      "required": [
        "name"
      ],
      "properties": {
        "name": {
          "type": "string"
        },
        "tag": {
          "type": "string"
        }
      }
    },
    "ErrorModel": {
      "type": "object",
      "required": [
        "code",
        "message"
      ],
      "properties": {
        "code": {
          "type": "integer",
          "format": "int32"
        },
        "message": {
          "type": "string"
        }
      }
    }
  }
}
`

func stringP(in string) *string {
	return &in
}

func TestSubscription(t *testing.T) {

	//
	// initialize central client
	// set your client id, privateKey, publicKey below
	//
	err := agent.Initialize(&config.CentralConfiguration{
		AgentType:              config.DiscoveryAgent,
		Mode:                   config.PublishToEnvironmentAndCatalog,
		TenantID:               "251204211014979",
		TeamName:               "Default Team",
		APICDeployment:         "prod",
		Environment:            "kong-mlo",
		URL:                    "https://apicentral.axway.com",
		PlatformURL:            "https://platform.axway.com",
		APIServerVersion:       "v1alpha1",
		TagsToPublish:          "mytag",
		AppendDataPlaneToTitle: false,
		Auth: &config.AuthConfiguration{
			URL:            "https://login.axway.com/auth",
			Realm:          "Broker",
			ClientID:       "DOSA_3c5fdc87453f43a597aa117c650b83ee",    // change
			PrivateKey:     "/home/look/projects/kong/private_key.pem", // change
			PublicKey:      "/home/look/projects/kong/public_key.pem",  // change
			PrivateKeyData: "",
			PublicKeyData:  "",
			KeyPwd:         "",
			Timeout:        0,
		},
		PollInterval:              10,
		ProxyURL:                  "",
		SubscriptionConfiguration: config.NewSubscriptionConfig(),
	})

	/*
		//
		// initialize kong client
		// set your kong url below
		//

		kURL := "http://localhost:8001" // change

		k, err := kong.NewClient(&kURL, &http.Client{})
		if err != nil {
			t.Fatalf("Failed due: %s", err)
		}

		//
		// create test kong resources
		//
		name := stringP("petstore")
		// create the route and service
		svc, err := k.Services.Create(context.TODO(), &kong.Service{
			Name:     name,
			Host:     stringP("petstore.swagger.io"),
			Path:     stringP("/v2"),
			Protocol: stringP("http"),
		})
		if err != nil {
			if e, ok := err.(*kong.APIError); !ok || (ok && e.Code() != 409) {
				t.Fatalf("Failed due: %s", err)
			}
			svc, err = k.Services.Get(context.TODO(), name)
			if err != nil {
				t.Fatalf("Failed due: %s", err)
			}
		}
		route, err := k.Routes.CreateInService(context.TODO(), svc.ID, &kong.Route{
			Name:  name,
			Hosts: []*string{stringP("localhost")},
			Paths: []*string{stringP("/" + *name)},
		})
		if err != nil {
			if e, ok := err.(*kong.APIError); !ok || (ok && e.Code() != 409) {
				t.Fatalf("Failed due: %s", err)
			}
			route, err = k.Routes.Get(context.TODO(), name)
			if err != nil {
				t.Fatalf("Failed due: %s", err)
			}
		}
		_, err = k.Plugins.Create(context.TODO(), &kong.Plugin{
			Name:  stringP("key-auth"),
			Route: route,
		})
		if err != nil {
			if e, ok := err.(*kong.APIError); !ok || (ok && e.Code() != 409) {
				t.Fatalf("Failed due: %s", err)
			}
		}
	*/
	//
	// this should happen as the agent is starting up
	//

	sm := subscription.New(logrus.StandardLogger(), agent.GetCentralClient(), k)

	// register schemas
	for _, schema := range sm.Schemas() {
		err := agent.GetCentralClient().RegisterSubscriptionSchema(schema)
		if err != nil {
			t.Fatalf("Failed due: %s", err)
		}
	}

	// register validator and handlers
	agent.GetCentralClient().GetSubscriptionManager().RegisterValidator(sm.ValidateSubscription)
	// register validator and handlers
	agent.GetCentralClient().GetSubscriptionManager().RegisterProcessor(apic.SubscriptionApproved, sm.ProcessSubscribe)

	// start polling for subscriptions
	agent.GetCentralClient().GetSubscriptionManager().Start()

	//
	// this should happen for each service
	//

	sb := apic.ServiceBody{
		NameToPush:        "testsvc",
		APIName:           "mytestapi",
		RestAPIID:         "myrestapid",
		URL:               "https://myapi.com",
		Version:           "v1",
		Swagger:           []byte(swagger),
		Tags:              map[string]interface{}{"tag": nil},
		AgentMode:         config.PublishToEnvironmentAndCatalog,
		CreatedBy:         "me",
		Status:            apic.PublishedStatus,
		ServiceAttributes: map[string]string{"attr": "value"},
	}

	//
	// Get the routeID and serviceID and pass them in here along with the ServiceBody
	// If the routeID/serviceID have an key-auth plugin defined the right subscription
	// will be defined
	//
	/*	err = sm.PopulateSubscriptionParameters(*route.ID, *svc.ID, &sb)
		if err != nil {
			t.Fatalf("Failed due: %s", err)
		}
	*/

	sb.AuthPolicy = apic.Apikey
	sb.SubscriptionName = apikey.Name

	_, err = agent.GetCentralClient().PublishService(sb)
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	// wait forever
	for {
		time.Sleep(time.Second)
	}
}
