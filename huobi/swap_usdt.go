package huobi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	goex "github.com/primitivelab/goexchange"
)

// SwapUsdt huobi coin margined contract
type SwapUsdt struct {
	httpClient *http.Client
	baseURL    string
	accountId  string
	accessKey  string
	secretKey  string
}

// NewSwapUsdt new instance
func NewSwapUsdt(client *http.Client, baseURL, apiKey, secretKey, accountID string) *SwapUsdt {
	instance := new(SwapUsdt)
	if baseURL == "" {
		instance.baseURL = "https://api.hbdm.com"
	} else {
		instance.baseURL = baseURL
	}
	instance.httpClient = client
	instance.accessKey = apiKey
	instance.secretKey = secretKey
	instance.accountId = accountID
	return instance
}

// NewSwapUsdtWithConfig new instance with config struct
func NewSwapUsdtWithConfig(config *goex.APIConfig) *SwapUsdt {
	instance := new(SwapUsdt)
	if config.Endpoint == "" {
		instance.baseURL = "https://api.hbdm.com"
	} else {
		instance.baseURL = config.Endpoint
	}
	instance.httpClient = config.HttpClient
	instance.accessKey = config.ApiKey
	instance.secretKey = config.ApiSecretKey
	instance.accountId = config.AccountId
	return instance
}

// GetExchangeName get exchange name
func (swap *SwapUsdt) GetExchangeName() string {
	return goex.EXCHANGE_HUOBI
}

// GetContractList exchange contract list
func (swap *SwapUsdt) GetContractList() interface{} {
	params := &url.Values{}
	return swap.httpGet("/linear-swap-api/v1/swap_contract_info", params, false)
}

// GetDepth exchange depth data
func (swap *SwapUsdt) GetDepth(symbol goex.Symbol, size int, options map[string]string) map[string]interface{} {
	params := &url.Values{}
	params.Set("contract_code", swap.getSymbol(symbol))
	if depthType, ok := options["type"]; ok {
		params.Set("type", depthType)
	} else {
		params.Set("type", "step0")
	}

	result := swap.httpGet("/linear-swap-ex/market/depth", params, false)
	if result["code"] != 0 {
		return result
	}
	result["data"] = result["data"].(map[string]interface{})["tick"]
	return result
}

// GetTicker exchange ticker data
func (swap *SwapUsdt) GetTicker(symbol goex.Symbol) interface{} {
	params := &url.Values{}
	params.Set("contract_code", swap.getSymbol(symbol))
	result := swap.httpGet("/linear-swap-ex/market/detail/merged", params, false)
	if result["code"] != 0 {
		return result
	}
	result["data"] = result["data"].(map[string]interface{})["tick"]
	return result
}

// GetKline exchange kline data
func (swap *SwapUsdt) GetKline(symbol goex.Symbol, period, size int, options map[string]string) interface{} {
	params := &url.Values{}
	params.Set("contract_code", swap.getSymbol(symbol))
	periodStr, ok := klinePeriod[period]
	if !ok {
		periodStr = "1min"
	}
	params.Set("period", periodStr)
	if size != 0 {
		params.Set("size", strconv.Itoa(size))
	}
	if from, ok := options["from"]; ok {
		params.Set("from", from)
	}
	if to, ok := options["to"]; ok {
		params.Set("to", to)
	}

	result := swap.httpGet("/linear-swap-ex/market/history/kline", params, false)
	if result["code"] != 0 {
		return result
	}
	result["data"] = result["data"].(map[string]interface{})["data"]
	return result
}

// GetTrade exchange trade order data
func (swap *SwapUsdt) GetTrade(symbol goex.Symbol, size int, options map[string]string) interface{} {
	params := &url.Values{}
	params.Set("contract_code", swap.getSymbol(symbol))
	if size != 0 {
		params.Set("size", strconv.Itoa(size))
	}
	result := swap.httpGet("/linear-swap-ex/market/history/trade", params, false)
	if result["code"] != 0 {
		return result
	}
	result["data"] = result["data"].(map[string]interface{})["data"]
	return result
}

// GetPremiumIndex exchange index price& market price & funding rate
func (swap *SwapUsdt) GetPremiumIndex(symbol goex.Symbol) interface{} {
	params := &url.Values{}
	if symbol.CoinFrom != "" {
		params.Set("contract_code", swap.getSymbol(symbol))
	}
	result := swap.httpGet("/linear-swap-api/v1/swap_index", params, false)
	if result["code"] != 0 {
		return result
	}
	fmt.Println(result)
	result["data"] = result["data"].(map[string]interface{})["data"]
	return result
}

// HTTPRequest request url
func (swap *SwapUsdt) HTTPRequest(requestURL, method string, options interface{}, signed bool) interface{} {
	method = strings.ToUpper(method)
	params := &url.Values{}
	mapOptions := options.(map[string]string)
	for key, val := range mapOptions {
		params.Set(key, val)
	}
	switch method {
	case goex.HTTP_GET:
		return swap.httpGet(requestURL, params, signed)
	case goex.HTTP_POST:
		return swap.httpPost(requestURL, params, signed)
	}
	return nil
}

// httpGet Get request method
func (swap *SwapUsdt) httpGet(url string, params *url.Values, signed bool) map[string]interface{} {
	var responseMap goex.HttpClientResponse
	sign := ""
	if signed {
		sign = swap.sign(goex.HTTP_GET, url, params)
	}

	requestURL := swap.baseURL + url
	if params != nil {
		requestURL = requestURL + "?" + params.Encode()
		if sign != "" {
			requestURL = requestURL + "&Signature=" + sign
		}
	}
	responseMap = goex.HttpGet(swap.httpClient, requestURL)
	return swap.handlerResponse(&responseMap)
}

// httpPost Post request method
func (swap *SwapUsdt) httpPost(path string, params *url.Values, signed bool) map[string]interface{} {
	var responseMap goex.HttpClientResponse

	signParams := &url.Values{}
	sign := swap.sign(goex.HTTP_POST, path, signParams)
	requestURL := swap.baseURL + path + "?" + signParams.Encode() + "&Signature=" + sign

	bodyMap := map[string]string{}
	for key, item := range *params {
		bodyMap[key] = item[0]
	}
	jsonBody, _ := json.Marshal(bodyMap)
	responseMap = goex.HttpPostWithJson(swap.httpClient, requestURL, string(jsonBody), map[string]string{})
	return swap.handlerResponse(&responseMap)
}

// httpPostBatch Post request method
func (swap *SwapUsdt) httpPostBatch(path string, params interface{}, signed bool) map[string]interface{} {
	var responseMap goex.HttpClientResponse
	jsonBody, _ := json.Marshal(params)

	signParams := &url.Values{}
	sign := swap.sign(goex.HTTP_POST, path, signParams)
	requestURL := swap.baseURL + path + "?" + signParams.Encode() + "&Signature=" + sign
	responseMap = goex.HttpPostWithJson(swap.httpClient, requestURL, string(jsonBody), map[string]string{})
	return swap.handlerResponse(&responseMap)
}

// handlerResponse Handler response data format
func (swap *SwapUsdt) handlerResponse(responseMap *goex.HttpClientResponse) map[string]interface{} {
	retData := make(map[string]interface{})

	retData["code"] = responseMap.Code
	retData["st"] = responseMap.St
	retData["et"] = responseMap.Et
	if responseMap.Code != 0 {
		retData["msg"] = responseMap.Msg
		retData["error"] = responseMap.Error
		return retData
	}

	var bodyDataMap map[string]interface{}
	err := json.Unmarshal(responseMap.Data, &bodyDataMap)
	if err != nil {
		retData["code"] = goex.JsonUnmarshalError.Code
		retData["msg"] = goex.JsonUnmarshalError.Msg
		retData["error"] = err.Error()
		return retData
	}

	if status, ok := bodyDataMap["status"]; ok && status.(string) != "ok" {
		retData["code"] = goex.ExchangeError.Code
		retData["msg"] = goex.ExchangeError.Msg
		if msg, ok := bodyDataMap["err-msg"]; ok {
			retData["error"] = msg.(string)
			return retData
		}

		if msg, ok := bodyDataMap["err_msg"]; ok {
			retData["error"] = msg.(string)
			return retData
		}
		return retData
	}
	if code, ok := bodyDataMap["code"]; ok && code.(float64) != 200 {
		retData["code"] = goex.ExchangeError.Code
		retData["msg"] = goex.ExchangeError.Msg
		retData["error"] = bodyDataMap["message"].(string)
		return retData
	}

	retData["data"] = bodyDataMap
	return retData
}

// sign signature method
func (swap *SwapUsdt) sign(method, path string, params *url.Values) string {
	host, _ := url.Parse(swap.baseURL)
	params.Set("AccessKeyId", swap.accessKey)
	params.Set("SignatureMethod", "HmacSHA256")
	params.Set("SignatureVersion", "2")
	params.Set("Timestamp", goex.GetNowUtcTime())
	parameters := params.Encode()

	var sb strings.Builder
	sb.WriteString(method)
	sb.WriteString("\n")
	sb.WriteString(host.Host)
	sb.WriteString("\n")
	sb.WriteString(path)
	sb.WriteString("\n")
	sb.WriteString(parameters)

	sign, _ := goex.HmacSha256Base64Signer(sb.String(), swap.secretKey)
	return sign
}

// getSymbol format symbol method
func (swap SwapUsdt) getSymbol(symbol goex.Symbol) string {
	return symbol.ToUpper().ToSymbol("-")
}
