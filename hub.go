package main

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered customers. HashKey(Client)
	customers map[string]*Customer

	// Registered agents. HashKey(Agent)
	agents map[string]*Agent

	// Register requests from the customers.
	cRegister chan *Customer

	// Register requests from the agents.
	aRegister chan *Agent

	// Unregister requests from customers.
	cUnregister chan *Customer

	// Unregister requests from agnents.
	aUnregister chan *Agent

	cusMapToAgent map[*Customer]*Agent

	agentMapToCus map[*Agent]*Customer

	availableAgents []*Agent
}

func newHub() *Hub {
	return &Hub{
		cRegister:   make(chan *Customer),
		aRegister:   make(chan *Agent),
		cUnregister: make(chan *Customer),
		aUnregister: make(chan *Agent),
		customers:   make(map[string]*Customer),
		agents:      make(map[string]*Agent),
		availableAgents: make([]*Agent, 0),
		cusMapToAgent: make(map[*Customer]*Agent),
		agentMapToCus: make(map[*Agent]*Customer),
	}
}

func (h *Hub) nextAvailableAgent() *Agent{
	if len(h.availableAgents) > 0 {
		a := h.availableAgents[0]
		h.availableAgents = h.availableAgents[1:]
		return a
	}
	return nil
}

func (h *Hub) run() {
	for {
		select {
		case customer := <-h.cRegister:
			h.customers[customer.HashKey()] = customer
		case agent := <-h.aRegister:
			h.agents[agent.HashKey()] = agent
		case customer := <-h.cUnregister:
			cusKey := customer.HashKey()
			agent := h.cusMapToAgent[customer]
			agentKey := agent.HashKey()
			if _, ok := h.customers[cusKey]; ok {
				delete(h.customers, cusKey)
				delete(h.agents, agentKey)
				delete(h.cusMapToAgent, customer)
				delete(h.agentMapToCus, agent)
				close(customer.recv)
				close(agent.recv)
			}
		case agent := <-h.aUnregister:
			agentKey := agent.HashKey()
			customer := h.agentMapToCus[agent]
			cusKey := customer.HashKey()
			if _, ok := h.customers[agentKey]; ok {
				delete(h.customers, cusKey)
				delete(h.agents, agentKey)
				delete(h.cusMapToAgent, customer)
				delete(h.agentMapToCus, agent)
				close(customer.recv)
				close(agent.recv)
			}
		}
	}
}
