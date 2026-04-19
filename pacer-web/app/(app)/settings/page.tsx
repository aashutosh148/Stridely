'use client';

import { useState } from 'react';
import { Settings as SettingsIcon, User, Bell, Link as LinkIcon, Shield, Save } from 'lucide-react';
import { useUser } from '@/hooks/useUser';

export default function SettingsPage() {
  const { user } = useUser();
  const [activeTab, setActiveTab] = useState('profile');
  const [isSaving, setIsSaving] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  // Mock form state - would connect to actual backend
  const [profileData, setProfileData] = useState({
    name: user?.name || '',
    email: user?.email || '',
    weight: user?.weight_kg || '',
    maxHr: user?.max_hr || '',
    restingHr: user?.resting_hr || '',
  });

  const [notifications, setNotifications] = useState({
    workoutReminders: true,
    weeklyReport: true,
    achievementAlerts: true,
    coachMessages: true,
  });

  const handleSaveProfile = async () => {
    setIsSaving(true);
    setSaveSuccess(false);
    
    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 1000));
    
    setIsSaving(false);
    setSaveSuccess(true);
    setTimeout(() => setSaveSuccess(false), 3000);
  };

  const tabs = [
    { id: 'profile', name: 'Profile', icon: User },
    { id: 'notifications', name: 'Notifications', icon: Bell },
    { id: 'integrations', name: 'Integrations', icon: LinkIcon },
    { id: 'privacy', name: 'Privacy', icon: Shield },
  ];

  return (
    <div className="space-y-6">
      {/* Dark Hero Header */}
      <div className="relative overflow-hidden rounded-lg bg-[#161b26] p-8 border border-gray-800">
        <div className="relative z-10">
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-blue-600">
              <SettingsIcon className="h-6 w-6 text-white" />
            </div>
            <div>
              <h1 className="text-3xl font-bold text-gray-100">Settings</h1>
              <p className="text-sm text-gray-400">Manage your account and preferences</p>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="overflow-hidden rounded-lg border border-gray-800 bg-[#161b26]">
        <div className="border-b border-gray-800 bg-[#1e2530] px-6 py-4">
          <nav className="flex space-x-2">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-all ${
                  activeTab === tab.id
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-400 hover:bg-[#1e2530] hover:text-gray-100'
                }`}
              >
                <tab.icon className="h-4 w-4" />
                {tab.name}
              </button>
            ))}
          </nav>
        </div>

        <div className="p-6">
          {/* Profile Tab */}
          {activeTab === 'profile' && (
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-bold text-gray-100">Personal Information</h3>
                <p className="text-sm text-gray-400">Update your personal details and athletic profile</p>
              </div>

              <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium text-gray-300">Full Name</label>
                  <input
                    type="text"
                    value={profileData.name}
                    onChange={(e) => setProfileData({ ...profileData, name: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-gray-800 bg-[#1e2530] px-4 py-2 text-sm text-gray-100 transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/20 focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300">Email Address</label>
                  <input
                    type="email"
                    value={profileData.email}
                    onChange={(e) => setProfileData({ ...profileData, email: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-gray-800 bg-[#1e2530] px-4 py-2 text-sm text-gray-100 transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/20 focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300">Weight (kg)</label>
                  <input
                    type="number"
                    value={profileData.weight}
                    onChange={(e) => setProfileData({ ...profileData, weight: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-gray-800 bg-[#1e2530] px-4 py-2 text-sm text-gray-100 transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/20 focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300">Max Heart Rate (bpm)</label>
                  <input
                    type="number"
                    value={profileData.maxHr}
                    onChange={(e) => setProfileData({ ...profileData, maxHr: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-gray-800 bg-[#1e2530] px-4 py-2 text-sm text-gray-100 transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/20 focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300">Resting Heart Rate (bpm)</label>
                  <input
                    type="number"
                    value={profileData.restingHr}
                    onChange={(e) => setProfileData({ ...profileData, restingHr: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-gray-800 bg-[#1e2530] px-4 py-2 text-sm text-gray-100 transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/20 focus:outline-none"
                  />
                </div>
              </div>

              <div className="flex items-center gap-3">
                <button
                  onClick={handleSaveProfile}
                  disabled={isSaving}
                  className="flex items-center gap-2 rounded-lg bg-blue-600 px-6 py-3 text-sm font-bold text-white transition hover:bg-blue-500 disabled:opacity-50"
                >
                  <Save className="h-4 w-4" />
                  {isSaving ? 'Saving...' : 'Save Changes'}
                </button>
                {saveSuccess && (
                  <span className="text-sm font-medium text-green-400">Changes saved successfully!</span>
                )}
              </div>
            </div>
          )}

          {/* Notifications Tab */}
          {activeTab === 'notifications' && (
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-bold text-gray-100">Notification Preferences</h3>
                <p className="text-sm text-gray-400">Choose what updates you want to receive</p>
              </div>

              <div className="space-y-4">
                {Object.entries({
                  workoutReminders: 'Daily workout reminders',
                  weeklyReport: 'Weekly training summary',
                  achievementAlerts: 'Achievement and milestone alerts',
                  coachMessages: 'Messages from your AI coach',
                }).map(([key, label]) => (
                  <div key={key} className="flex items-center justify-between rounded-lg bg-[#1e2530] border border-gray-800 p-4">
                    <span className="text-sm font-medium text-gray-100">{label}</span>
                    <button
                      onClick={() => setNotifications({ ...notifications, [key]: !notifications[key as keyof typeof notifications] })}
                      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                        notifications[key as keyof typeof notifications] ? 'bg-blue-600' : 'bg-gray-700'
                      }`}
                    >
                      <span
                        className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                          notifications[key as keyof typeof notifications] ? 'translate-x-6' : 'translate-x-1'
                        }`}
                      />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Integrations Tab */}
          {activeTab === 'integrations' && (
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-bold text-gray-100">Connected Services</h3>
                <p className="text-sm text-gray-400">Manage your third-party integrations</p>
              </div>

              <div className="space-y-4">
                {['Strava', 'Garmin Connect', 'Apple Health', 'Google Fit'].map((service) => (
                  <div key={service} className="flex items-center justify-between rounded-lg border border-gray-800 bg-[#1e2530] p-5">
                    <div>
                      <p className="font-semibold text-gray-100">{service}</p>
                      <p className="text-xs text-gray-500">
                        {service === 'Strava' ? 'Connected' : 'Not connected'}
                      </p>
                    </div>
                    <button
                      className={`rounded-lg px-4 py-2 text-sm font-semibold transition ${
                        service === 'Strava'
                          ? 'bg-red-900/50 text-red-400 border border-red-800 hover:bg-red-900/70'
                          : 'bg-blue-900/50 text-blue-400 border border-blue-800 hover:bg-blue-900/70'
                      }`}
                    >
                      {service === 'Strava' ? 'Disconnect' : 'Connect'}
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Privacy Tab */}
          {activeTab === 'privacy' && (
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-bold text-gray-100">Privacy & Security</h3>
                <p className="text-sm text-gray-400">Control your data and privacy settings</p>
              </div>

              <div className="space-y-4">
                <div className="rounded-lg border border-gray-800 bg-[#1e2530] p-5">
                  <h4 className="font-semibold text-gray-100">Data Export</h4>
                  <p className="mt-1 text-sm text-gray-400">Download all your training data</p>
                  <button className="mt-3 rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-blue-500">
                    Request Export
                  </button>
                </div>

                <div className="rounded-lg border border-gray-800 bg-[#1e2530] p-5">
                  <h4 className="font-semibold text-gray-100">Password</h4>
                  <p className="mt-1 text-sm text-gray-400">Change your account password</p>
                  <button className="mt-3 rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-blue-500">
                    Update Password
                  </button>
                </div>

                <div className="rounded-lg border border-red-800 bg-red-900/20 p-5">
                  <h4 className="font-semibold text-red-400">Delete Account</h4>
                  <p className="mt-1 text-sm text-red-300/80">Permanently delete your account and all data</p>
                  <button className="mt-3 rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-red-500">
                    Delete Account
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
