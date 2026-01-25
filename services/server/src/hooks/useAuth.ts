import { useSession, signIn, signUp, signOut } from "../lib/auth-client";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";

export function useAuth() {
  const { data: session, isPending } = useSession();
  const navigate = useNavigate();

  const handleSignIn = async (email: string, password: string) => {
    try {
      const result = await signIn.email({
        email,
        password,
      });

      if (result.error) {
        toast.error(result.error.message || "Failed to sign in");
        return { error: result.error };
      }

      toast.success("Signed in successfully");
      navigate("/dashboard");
      return { data: result.data };
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to sign in";
      toast.error(message);
      return { error: { message } };
    }
  };

  const handleSignUp = async (
    email: string,
    password: string,
    name: string
  ) => {
    try {
      const result = await signUp.email({
        email,
        password,
        name,
      });

      if (result.error) {
        toast.error(result.error.message || "Failed to sign up");
        return { error: result.error };
      }

      // Since email verification is required, show message and stay on page
      toast.success("Account created! Please check your email to verify your account.");
      // Don't navigate - user needs to verify email first
      return { data: result.data };
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to sign up";
      toast.error(message);
      return { error: { message } };
    }
  };

  const handleSignOut = async () => {
    try {
      await signOut();
      toast.success("Signed out successfully");
      navigate("/auth/sign-in");
    } catch (error) {
      toast.error("Failed to sign out");
    }
  };

  return {
    user: session?.user,
    session: session?.session,
    isPending,
    isAuthenticated: !!session?.user,
    signIn: handleSignIn,
    signUp: handleSignUp,
    signOut: handleSignOut,
  };
}
