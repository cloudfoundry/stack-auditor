package resources

type Orgs []plugin_models.GetOrgs_Model

func (o Orgs) Map() map[string]string {
	m := make(map[string]string)

	for _, org := range o {
		m[org.Guid] = org.Name
	}
	return m
}
