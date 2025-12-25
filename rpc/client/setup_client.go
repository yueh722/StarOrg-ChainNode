package client

import (
	"chain/lib"
	"chain/rpc/common"
	"chain/rpc/server"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

type SetupClient struct {
	client *rpc.Client
	nodeID string
}

var router = map[common.CmdType]func(SetupClient, *common.CommandEnvelope) error{
	common.CmdInitializeChainNode: SetupClient.handleInitChainNodeCmd,
	common.CmdAddChainNode:        SetupClient.handleAddChainNodeCmd,
	common.CmdControlNode:         SetupClient.handleControlNodeCmd,
}

func newSetupClient(client *rpc.Client) *SetupClient {
	return &SetupClient{client: client}
}

func (c *SetupClient) Update(client *rpc.Client) {
	c.client = client
}

func (c *SetupClient) SubscribeCommand() <-chan error {

	var errChan = make(chan error)

	setupService := &server.SetupService{}
	//ctx, cancel := context.WithCancel(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmdChan := make(chan common.CommandEnvelope, 1000)

	sub, err := c.client.Subscribe(
		ctx,
		setupService.GetNamesapceString(),
		cmdChan,
		setupService.GetSubscriptionName(),
		c.nodeID,
	)

	if err != nil {
		return errChan
	}

	go func() {
		defer func() {
			sub.Unsubscribe()
			errChan <- rpc.ErrClientQuit
			close(cmdChan)
		}()

		exitSub := false
		for !exitSub {
			select {
			case cmdEnvelope := <-cmdChan:
				log.Printf("command: %v", cmdEnvelope.Payload)
				err := c.handleCmd(&cmdEnvelope)
				if err != nil {
					log.Printf("failed to handle cmd: %v", err)
				}
			case errChan := <-sub.Err():
				log.Printf("subscription Err: %v", errChan)
				exitSub = true
				break
			}
		}
	}()

	return errChan
}

func (c SetupClient) handleCmd(cmd *common.CommandEnvelope) error {
	if cmd.NodeID != c.nodeID {
		return fmt.Errorf("nodeID mismatch, ignore")
	}

	err := router[cmd.Cmd](c, cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c SetupClient) handleInitChainNodeCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupInitializeChainNode{}
	err := json.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	nodeName := ""

	err = lib.InitializeChainNode(
		arg.GenesisTime,
		arg.NodeArray,
		arg.ChainID,
		arg.NodeID,
		"",
	)
	if err != nil {
		log.Printf("failed to init chain node id: %s, node name: %s, err: %v", cmd.NodeID, nodeName, err)
	}

	return err
}

func (c SetupClient) handleAddChainNodeCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupAddChainNode{}
	err := json.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	nodeName := ""

	err = lib.AddChainNode(
		arg.NodeArray,
		arg.GenesisDoc,
		arg.NodeID,
		"",
	)
	if err != nil {
		log.Printf("failed to add chain node id: %s, node name: %s, err: %v", cmd.NodeID, nodeName, err)
	}

	return err
}

func (c SetupClient) handleControlNodeCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupControlNode{}
	err := json.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	router := map[common.NodeAction]func(nodeID string) error{
		common.NodeStart:   lib.StartNode,
		common.NodeStop:    lib.StopNode,
		common.NodeRestart: lib.RestartNode,
	}

	routine, ok := router[arg.Action]
	if !ok {
		return fmt.Errorf("invalid action: %v", arg.Action)
	}

	err = routine(arg.NodeID)
	if err != nil {
		log.Printf("failed to %s chain node id: %s, node name: %s, err: %v", arg.Action, cmd.NodeID, err)
	}

	return err
}
