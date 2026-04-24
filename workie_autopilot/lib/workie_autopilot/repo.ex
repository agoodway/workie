defmodule WorkieAutopilot.Repo do
  use Ecto.Repo,
    otp_app: :workie_autopilot,
    adapter: Ecto.Adapters.SQLite3
end
