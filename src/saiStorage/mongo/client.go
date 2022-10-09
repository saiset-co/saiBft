package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/iamthe1whoknocks/saiStorage/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	Config config.Configuration
	Host   *mongo.Client
	Ctx    context.Context
}

type FindResult struct {
	Count  int64         `json:"count,omitempty"`
	Result []interface{} `json:"result,omitempty"`
}

type Options struct {
	Limit int64  `json:"limit"`
	Skip  int64  `json:"skip"`
	Sort  bson.M `json:"sort"`
	Count int64  `json:"count"`
}

func NewMongoClient(config config.Configuration) (Client, error) {
	var host *mongo.Client
	var hostErr error

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch config.Storage.Atlas {
	case false:
		{
			host, _ = mongo.NewClient(options.Client().ApplyURI(
				"mongodb://" + config.Storage.Host + ":" + config.Storage.Port + "/" + config.Storage.Database,
			))

			hostErr = host.Connect(ctx)
		}
	default:
		{
			host, hostErr = mongo.Connect(ctx, options.Client().ApplyURI(
				"mongodb+srv://"+config.Storage.User+":"+config.Storage.Pass+"@"+config.Storage.Host+"/"+config.Storage.Database+"?ssl=true&authSource=admin&retryWrites=true&w=majority",
			))
		}
	}

	client := Client{
		Ctx:    ctx,
		Config: config,
		Host:   host,
	}

	if hostErr != nil {
		return client, hostErr
	}

	return client, nil
}

func (c Client) GetCollection(collectionName string) *mongo.Collection {
	return c.Host.Database(c.Config.Storage.Database).Collection(collectionName)
}

func (c Client) FindOne(collectionName string, selector map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	collection := c.GetCollection(collectionName)
	selector = c.preprocessSelector(selector)
	cur, err := collection.Find(context.TODO(), selector)

	if err != nil {
		return result, err
	}

	defer cur.Close(context.TODO())

	for cur.Next(context.TODO()) {
		var elem map[string]interface{}
		decodeErr := cur.Decode(&elem)

		if decodeErr != nil {
			return result, decodeErr
		}

		result = elem
		break
	}

	if cursorErr := cur.Err(); cursorErr != nil {
		return result, cursorErr
	}

	return result, nil
}

func (c Client) Find(collectionName string, selector map[string]interface{}, inputOptions Options) (*FindResult, error) {
	findResult := &FindResult{}
	requestOptions := options.Find()

	if inputOptions.Count != 0 {
		collection := c.GetCollection(collectionName)
		selector = c.preprocessSelector(selector)
		count, err := collection.CountDocuments(context.TODO(), selector)
		if err != nil {
			return &FindResult{}, err
		}
		return &FindResult{
			Count: count,
		}, nil
	}

	if inputOptions.Sort != nil {
		requestOptions.SetSort(inputOptions.Sort)
	}

	if inputOptions.Skip != 0 {
		requestOptions.SetSkip(inputOptions.Skip)
	}

	if inputOptions.Limit != 0 {
		requestOptions.SetLimit(inputOptions.Limit)
	}

	collection := c.GetCollection(collectionName)
	selector = c.preprocessSelector(selector)

	cur, err := collection.Find(context.TODO(), selector, requestOptions)

	if err != nil {
		return &FindResult{}, err
	}

	defer cur.Close(context.TODO())

	for cur.Next(context.TODO()) {
		var elem map[string]interface{}
		decodeErr := cur.Decode(&elem)

		if decodeErr != nil {
			return &FindResult{}, decodeErr
		}

		findResult.Result = append(findResult.Result, elem)
	}

	if cursorErr := cur.Err(); cursorErr != nil {
		return findResult, cursorErr
	}

	return findResult, nil
}

func (c Client) Insert(collectionName string, doc interface{}) error {
	collection := c.GetCollection(collectionName)

	_, err := collection.InsertOne(context.TODO(), doc)
	if err != nil {
		return err
	}

	return nil
}

func (c Client) Update(collectionName string, selector map[string]interface{}, update interface{}) error {
	collection := c.GetCollection(collectionName)
	selector = c.preprocessSelector(selector)

	_, err := collection.UpdateMany(context.TODO(), selector, update)
	if err != nil {
		return err
	}

	return nil
}

func (c Client) Upsert(collectionName string, selector map[string]interface{}, update interface{}) error {
	collection := c.GetCollection(collectionName)
	requestOptions := options.Update().SetUpsert(true)
	selector = c.preprocessSelector(selector)

	_, err := collection.UpdateMany(context.TODO(), selector, update, requestOptions)
	if err != nil {
		return err
	}

	return nil
}

func (c Client) Remove(collectionName string, selector map[string]interface{}) error {
	collection := c.GetCollection(collectionName)
	selector = c.preprocessSelector(selector)

	_, err := collection.DeleteOne(context.TODO(), selector)
	if err != nil {
		return err
	}

	return nil
}

func (c Client) preprocessSelector(selector map[string]interface{}) map[string]interface{} {
	if selector["_id"] != nil {
		objID, err := primitive.ObjectIDFromHex(selector["_id"].(string))

		if err != nil {
			fmt.Println("Wrong objectId string")
			return selector
		}

		selector["_id"] = objID
	}

	return selector
}
