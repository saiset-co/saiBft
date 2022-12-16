package internal

import (
	"encoding/json"
	"errors"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// check if node is a validator
// update parameters, if no validators was found in config
func (s *InternalService) setValidators(storageToken string) error {
	err, result := s.Storage.Get(ParametersCol, bson.M{}, bson.M{}, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("process - setValidator - get parameters", zap.Error(err))
		return err
	}

	if len(result) == 2 {
		s.GlobalService.Logger.Debug("process - setValidator - get parameters - emtpy result from parameter collection")
		validators, err := s.getValidatorsFromConfig()
		if err != nil {
			s.GlobalService.Logger.Debug("process - setValidator - get parameters from config", zap.Error(err))
			return err
		}
		err, _ = s.Storage.Put(ParametersCol, models.Parameters{
			Validators:     validators,
			IsBoorstrapped: false,
		}, storageToken)
		if err != nil {
			s.GlobalService.Logger.Debug("process - setValidator - get parameters - put parameters", zap.Error(err))
			return err
		}

		s.Validators = validators
		return nil
	}

	data, err := utils.ExtractResult(result)
	if err != nil {
		return err
	}

	params := models.Parameters{}

	err = json.Unmarshal(data, &params)
	if err != nil {
		Service.GlobalService.Logger.Error("process - setValidator - unmarshal", zap.Error(err))
		return err
	}

	// validators was not found from parameters col, check config and update
	if params.Validators == nil {
		validators, err := s.getValidatorsFromConfig()
		if err != nil {
			s.GlobalService.Logger.Debug("process - setValidator - get parameters from config", zap.Error(err))
			return err
		}

		err, _ = s.Storage.Update(ParametersCol, bson.M{}, params, storageToken)
		if err != nil {
			Service.GlobalService.Logger.Error("process - setValidator - update validators", zap.Error(err))
			return err
		}
		s.Validators = validators
		return nil
	}

	s.Validators = params.Validators

	return nil
}

func (s *InternalService) getValidatorsFromConfig() ([]string, error) {
	validatorsInterface, ok := s.GlobalService.Configuration["validators"].([]interface{})
	if !ok {
		s.GlobalService.Logger.Error("process -  - wrong type of validators value from config")
		return nil, errors.New("wrong type of validators value in config")
	}

	validators := make([]string, 0)

	for _, validator := range validatorsInterface {
		validators = append(validators, validator.(string))
	}
	return validators, nil
}
