package policies

// Primary app roles
const (
	RoleDoctor    = "doctor"
	RoleDeveloper = "developer"
	RolePatient   = "patient"
)

// Simple allow-lists that you can expand later
func IsDoctor(role any) bool {
	s, _ := role.(string)
	return s == RoleDoctor
}

func IsDeveloper(role any) bool {
	s, _ := role.(string)
	return s == RoleDeveloper
}
