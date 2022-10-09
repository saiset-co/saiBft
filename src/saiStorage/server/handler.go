package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/iamthe1whoknocks/saiStorage/mongo"
	"github.com/iamthe1whoknocks/saiStorage/utils"
	"go.mongodb.org/mongo-driver/bson"
)

func (s Server) handleWebSocketRequest(msg []byte) {

}

type jsonRequestType struct {
	Collection string        `json:"collection"`
	Select     bson.M        `json:"select"`
	Options    mongo.Options `json:"options"`
	Data       bson.M        `json:"data"`
}

func (s Server) handleServerRequest(w http.ResponseWriter, r *http.Request) {
	client, err := mongo.NewMongoClient(s.Config)

	if err != nil {
		fmt.Println("Could not connect to the mongo server:", err)
	}

	switch r.URL.Path {
	case "/get":
		{
			s.get(client, w, r)
		}
	case "/save":
		{
			s.save(client, w, r)
		}
	case "/update":
		{
			s.update(client, w, r)
		}
	case "/upsert":
		{
			s.upsert(client, w, r)
		}
	case "/remove":
		{
			s.remove(client, w, r)
		}
	}

	client.Host.Disconnect(context.Background())
}

func (s Server) get(client mongo.Client, w http.ResponseWriter, r *http.Request) {
	var request jsonRequestType
	decoder := json.NewDecoder(r.Body)
	decoderErr := decoder.Decode(&request)

	if decoderErr != nil {
		fmt.Printf("Wrong JSON: %v", decoderErr)
		return
	}

	result, mongoErr := client.Find(request.Collection, request.Select, request.Options)

	if mongoErr != nil {
		fmt.Println("Mongo error:", mongoErr)
		return
	}

	_, writeErr := w.Write(utils.ConvertInterfaceToJson(result))

	if writeErr != nil {
		fmt.Println("Write error:", writeErr)
		return
	}
}

func (s Server) save(client mongo.Client, w http.ResponseWriter, r *http.Request) {
	var request jsonRequestType
	decoder := json.NewDecoder(r.Body)
	decoderErr := decoder.Decode(&request)

	if decoderErr != nil {
		fmt.Printf("Wrong JSON: %v", decoderErr)
		return
	}

	mongoErr := client.Insert(request.Collection, request.Data)

	if mongoErr != nil {
		fmt.Println("Mongo error:", mongoErr)
		return
	}

	_, writeErr := w.Write(utils.ConvertInterfaceToJson(bson.M{"Status": "Ok"}))

	if writeErr != nil {
		fmt.Println("Write error:", writeErr)
		return
	}
}

func (s Server) update(client mongo.Client, w http.ResponseWriter, r *http.Request) {
	var request jsonRequestType
	decoder := json.NewDecoder(r.Body)
	decoderErr := decoder.Decode(&request)

	if decoderErr != nil {
		fmt.Printf("Wrong JSON: %v", decoderErr)
		return
	}

	mongoErr := client.Update(request.Collection, request.Select, bson.M{"$set": request.Data})

	if mongoErr != nil {
		fmt.Println("Mongo error:", mongoErr)
		return
	}

	_, writeErr := w.Write(utils.ConvertInterfaceToJson(bson.M{"Status": "Ok"}))

	if writeErr != nil {
		fmt.Println("Write error:", writeErr)
		return
	}
}

func (s Server) upsert(client mongo.Client, w http.ResponseWriter, r *http.Request) {
	var request jsonRequestType
	decoder := json.NewDecoder(r.Body)
	decoderErr := decoder.Decode(&request)

	if decoderErr != nil {
		fmt.Printf("Wrong JSON: %v", decoderErr)
		return
	}

	mongoErr := client.Upsert(request.Collection, request.Select, request.Data)

	if mongoErr != nil {
		fmt.Println("Mongo error:", mongoErr)
		return
	}

	_, writeErr := w.Write(utils.ConvertInterfaceToJson(bson.M{"Status": "Ok"}))

	if writeErr != nil {
		fmt.Println("Write error:", writeErr)
		return
	}
}

func (s Server) remove(client mongo.Client, w http.ResponseWriter, r *http.Request) {
	var request jsonRequestType
	decoder := json.NewDecoder(r.Body)
	decoderErr := decoder.Decode(&request)

	if decoderErr != nil {
		fmt.Printf("Wrong JSON: %v", decoderErr)
		return
	}

	mongoErr := client.Remove(request.Collection, request.Select)

	if mongoErr != nil {
		fmt.Println("Mongo error:", mongoErr)
		return
	}

	_, writeErr := w.Write(utils.ConvertInterfaceToJson(bson.M{"Status": "Ok"}))

	if writeErr != nil {
		fmt.Println("Write error:", writeErr)
		return
	}
}
