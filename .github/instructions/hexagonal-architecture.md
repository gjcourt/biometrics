# Hexagonal Architecture Instruction

You MUST follow the Hexagonal Architecture (Ports and Adapters) design pattern for this project.
- **Domain**: Contains the core business logic and entities. It must not depend on any external frameworks or technologies.
- **Ports**: Interfaces defined in the domain layer that specify how the application interacts with the outside world (e.g., repositories, external services).
- **Adapters**: Implementations of the ports that interact with external systems (e.g., HTTP handlers, database repositories). Adapters depend on the domain, not the other way around.
- **App/Service**: Orchestrates the business logic by using the domain entities and ports.

Ensure that dependencies always point inwards towards the domain layer.
For more details, refer to the `docs/architecture.md` file.
