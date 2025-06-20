import { Header } from "@/components/layout/Header";
import { Outlet } from "react-router";

function App() {
  return (
    <div className="flex flex-col min-h-screen bg-background">
      {/* <Header /> */}
      <div className="flex-grow container mx-auto px-4 py-8">
        <Outlet />
      </div>
    </div>
  );
}

export default App;
