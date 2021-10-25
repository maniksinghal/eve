package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	webexteams "github.com/jbogarin/go-cisco-webex-teams/sdk"
	schema_db "github.com/maniksinghal/eve/timing-db"
)

func test_my_db(my_db *schema_db.Json_db) {
	var query = "Does Bifrost use MetaDX1 phy"
	var keywords = strings.Split(query, " ")
	responses, _, _ := schema_db.Query_database(keywords, my_db)
	for _, resp := range responses {
		fmt.Printf("The %s family %s card uses %s=%s on ports:%s, speeds:%s\n",
			resp.Family, resp.Pid, resp.Property, resp.Value, resp.Port_range, resp.Lane_speeds)
	}
}

func respond_to_query(message string, roomId string) {
	var keywords = strings.Split(message, " ")
	var response string
	responses, matched_families, matched_pids := schema_db.Query_database(keywords, my_db)

	for _, resp := range responses {
		this_resp := fmt.Sprintf("The %s family %s card uses %s=%s",
			resp.Family, resp.Pid, resp.Property, resp.Value)
		if len(resp.Port_range) > 0 {
			this_resp = this_resp + fmt.Sprintf(" on ports %s", resp.Port_range)
		}
		if len(resp.Lane_speeds) > 0 {
			this_resp = this_resp + fmt.Sprintf(" with speeds %s", resp.Lane_speeds)
		}

		response = response + this_resp + "\n"
	}

	negative_response := "Sorry, I didn't understand. Please try a simpler query"
	if len(responses) == 0 {
		if strings.Contains(message, "everything") {
			if len(matched_families) == 1 {
				response = my_db.Get_family_info(matched_families[0])
			} else if len(matched_pids) == 1 {
				response = my_db.Get_pid_info(matched_pids[0])
			} else {
				response = negative_response
			}
		} else {
			response = negative_response
		}
	}

	/*
		if len(responses) == 0 {
			response = "Sorry, I couldn't understand. Please try a simpler query"
		}
	*/

	if roomId != "test" {
		markDownMessage := &webexteams.MessageCreateRequest{
			Text:   response,
			RoomID: roomId,
			//ToPersonID: person_id,
		}

		newMarkDownMessage, _, err := Client.Messages.CreateMessage(markDownMessage)
		if err != nil {
			fmt.Println("Error sending message " + err.Error())
		}
		fmt.Println("POST:", newMarkDownMessage.ID, newMarkDownMessage.Markdown, newMarkDownMessage.Created)
	} else {
		fmt.Printf(response)
	}
}

// create a handler struct
type HttpHandler struct{}

// implement `ServeHTTP` method on `HttpHandler` struct
func (h HttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	fmt.Println("Bot received http request")

	/*
		for name, value := range req.Header {
			fmt.Printf("Header %s = %s\n", name, value)
		}
		fmt.Printf("Body: %s\n", req.Body)
	*/

	var result map[string]interface{}
	bodyBytes, _ := io.ReadAll(req.Body)

	json.Unmarshal(bodyBytes, &result)
	/*
		for name, value := range result {
			fmt.Printf("Body %s = %s\n", name, value)
		}
	*/

	_, exists := result["data"]
	if !exists {
		fmt.Println("Did not receive a valid message")
		return
	}

	var data map[string]interface{}
	data = result["data"].(map[string]interface{})
	_, exists = data["id"]
	if !exists {
		fmt.Println("Did not receive valid data in message")
		return
	}

	_, exists = data["roomId"]
	if !exists {
		fmt.Println("Did not receive source of the message")
		return
	}

	_, exists = data["personEmail"]
	if !exists {
		fmt.Println("Unable to identify the query sender")
		return
	}

	// Get the message text
	message_id := data["id"].(string)
	room_id := data["roomId"].(string)
	sender := data["personEmail"].(string)

	if sender == "maniktestbot@webex.bot" {
		// Its me only
		fmt.Println("Ignored processing of my own reply")
		return
	}

	//fmt.Println("Extracted message-id: " + message_id)
	htmlMessageGet, _, err := Client.Messages.GetMessage(message_id)
	if err != nil {
		fmt.Println("Failed to extract message " + err.Error())
	}

	fmt.Println("Received query " + htmlMessageGet.Text + " from " +
		data["personEmail"].(string))

	respond_to_query(htmlMessageGet.Text, room_id)

	// create response binary data
	resp_data := []byte("Hello World!") // slice of bytes
	// write `data` to response
	res.Write(resp_data)
}

var Client *webexteams.Client
var bot_id string

/*
 * List and delete all existing webhooks from
 * the bot
 * Webhook creation is done by hookbuster
 */
func cleanup_webhooks() {
	Client = webexteams.NewClient()
	bot_id = "MWU5ZGRmZmYtNDk4ZC00NTg1LWE0YmUtNTE2YjMyOWVhZGRjOWIxNmI2NmYtMGZj_PF84_1eb65fdf-9643-417f-9974-ad72cae0e10f"
	Client.SetAuthToken(bot_id)

	webhooksQueryParams := &webexteams.ListWebhooksQueryParams{
		Max: 10,
	}

	webhooks, _, err := Client.Webhooks.ListWebhooks(webhooksQueryParams)
	if err != nil {
		fmt.Println("Error listing the webhooks")
	}
	for id, webhook := range webhooks.Items {
		fmt.Println("GET:", id, webhook.ID, webhook.Name, webhook.TargetURL, webhook.Created)

		// DELETE webhooks/<ID>
		_, err := Client.Webhooks.DeleteWebhook(webhook.ID)
		if err != nil {
			fmt.Println("Error deleting webhook " + err.Error())
		} else {
			fmt.Println("Deleted existing webhook " + webhook.Name)
		}
	}

	// Webhooks are now created by hookbuster
	/*
		//myRoomID := ""                  // Change to your testing room
		//webHookURL := "https://abc.com" // Change this to your test URL
		webHookURL := "https://00c7-49-207-221-78.ngrok.io"

		// POST webhooks

		webhookRequest := &webexteams.WebhookCreateRequest{
			Name:      "Webhook - Test",
			TargetURL: webHookURL,
			Resource:  "messages",
			Event:     "created",
			//Filter:    "roomId=" + RoomID,
		}

		testWebhook, _, err := Client.Webhooks.CreateWebhook(webhookRequest)
		if err != nil {
			fmt.Println("Error creating webhook " + err.Error())
		}

		fmt.Println("POST:", testWebhook.ID, testWebhook.Name, testWebhook.TargetURL, testWebhook.Created)
	*/
}

var my_db *schema_db.Json_db

func test_my_bot() {
	excel_db := new(schema_db.Excel_db)
	err := excel_db.Parse_db("Timing PIDs.xlsx")
	if err != nil {
		fmt.Println(err)
		return
	}
	schema_db.Dump_db(excel_db)
}

func main() {

	test_my_bot()
	return

	my_db = new(schema_db.Json_db)
	schema_db.Parse_database(my_db, "pid_db.json")

	cleanup_webhooks()

	// create a new handler
	handler := HttpHandler{}

	// listen and serve
	fmt.Println("Listening on port 9000")
	http.ListenAndServe(":9000", handler)
	fmt.Println("Abrupt came out of listening!!")

}
