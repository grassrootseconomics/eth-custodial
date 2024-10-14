// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "termsOfService": "https://grassecon.org/pages/terms-and-conditions.html",
        "contact": {
            "name": "API Support",
            "url": "https://grassecon.org/pages/contact-us",
            "email": "devops@grassecon.org"
        },
        "license": {
            "name": "AGPL-3.0",
            "url": "https://www.gnu.org/licenses/agpl-3.0.en.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/account/create": {
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Create a new custodial account",
                "consumes": [
                    "*/*"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Account"
                ],
                "summary": "Create a new custodial account",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.OKResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    }
                }
            }
        },
        "/account/status/{address}": {
            "get": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Check a custodial account's status",
                "consumes": [
                    "*/*"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Account"
                ],
                "summary": "Check a custodial account's status",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Account address",
                        "name": "address",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.OKResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    }
                }
            }
        },
        "/otx/track/{trackingId}": {
            "get": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Track an OTX's (Origin transaction) chain status",
                "consumes": [
                    "*/*"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "OTX"
                ],
                "summary": "Track an OTX's (Origin transaction) chain status",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Tracking ID",
                        "name": "trackingId",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.OKResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    }
                }
            }
        },
        "/pool/deposit": {
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Pool deposit request",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Sign"
                ],
                "summary": "Pool deposit request",
                "parameters": [
                    {
                        "description": "Pool deposit request",
                        "name": "transferRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/api.PoolDepositRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.OKResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    }
                }
            }
        },
        "/pool/quote": {
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Get a pool swap quote",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Sign"
                ],
                "summary": "Get a pool swap quote",
                "parameters": [
                    {
                        "description": "Get a pool swap quote",
                        "name": "transferRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/api.PoolSwapRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.OKResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    }
                }
            }
        },
        "/pool/swap": {
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Pool swap request",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Sign"
                ],
                "summary": "Pool swap request",
                "parameters": [
                    {
                        "description": "Pool swap request",
                        "name": "transferRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/api.PoolSwapRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.OKResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    }
                }
            }
        },
        "/system": {
            "get": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Get the current system information",
                "consumes": [
                    "*/*"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "System"
                ],
                "summary": "Get the current system information",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.OKResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    }
                }
            }
        },
        "/token/transfer": {
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Sign a token transfer request",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Sign"
                ],
                "summary": "Sign a token transfer request",
                "parameters": [
                    {
                        "description": "Transfer request",
                        "name": "transferRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/api.TransferRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.OKResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.ErrResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "api.ErrResponse": {
            "type": "object",
            "properties": {
                "description": {
                    "type": "string"
                },
                "errorCode": {
                    "type": "string"
                },
                "ok": {
                    "type": "boolean"
                }
            }
        },
        "api.OKResponse": {
            "type": "object",
            "properties": {
                "description": {
                    "type": "string"
                },
                "ok": {
                    "type": "boolean"
                },
                "result": {
                    "type": "object",
                    "additionalProperties": {}
                }
            }
        },
        "api.PoolDepositRequest": {
            "type": "object",
            "required": [
                "amount",
                "from",
                "poolAddress",
                "tokenAddress"
            ],
            "properties": {
                "amount": {
                    "type": "string"
                },
                "from": {
                    "type": "string"
                },
                "poolAddress": {
                    "type": "string"
                },
                "tokenAddress": {
                    "type": "string"
                }
            }
        },
        "api.PoolSwapRequest": {
            "type": "object",
            "required": [
                "amount",
                "from",
                "fromTokenAddress",
                "poolAddress",
                "toTokenAddress"
            ],
            "properties": {
                "amount": {
                    "type": "string"
                },
                "from": {
                    "type": "string"
                },
                "fromTokenAddress": {
                    "type": "string"
                },
                "poolAddress": {
                    "type": "string"
                },
                "toTokenAddress": {
                    "type": "string"
                }
            }
        },
        "api.TransferRequest": {
            "type": "object",
            "required": [
                "amount",
                "from",
                "to",
                "tokenAddress"
            ],
            "properties": {
                "amount": {
                    "type": "string"
                },
                "from": {
                    "type": "string"
                },
                "to": {
                    "type": "string"
                },
                "tokenAddress": {
                    "type": "string"
                }
            }
        }
    },
    "securityDefinitions": {
        "ApiKeyAuth": {
            "description": "Service API Key",
            "type": "apiKey",
            "name": "X-GE-KEY",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "2.0",
	Host:             "localhost:5003",
	BasePath:         "/api/v2",
	Schemes:          []string{},
	Title:            "ETH Custodial API",
	Description:      "Interact with the Grassroots Economics Custodial API",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
