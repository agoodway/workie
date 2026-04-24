defmodule WorkieAutopilotWeb.Router do
  use WorkieAutopilotWeb, :router

  pipeline :api do
    plug :accepts, ["json"]
  end

  scope "/api", WorkieAutopilotWeb do
    pipe_through :api
  end
end
