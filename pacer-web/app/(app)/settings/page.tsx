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
      {/* Gradient Hero Header */}
      <div className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500 p-8 shadow-lg">
        <div className="absolute inset-0 bg-black/5"></div>
        <div className="relative z-10">
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-white/20 backdrop-blur-sm">
              <SettingsIcon className="h-6 w-6 text-white" />
            </div>
            <div>
              <h1 className="text-3xl font-bold text-white">Settings</h1>
              <p className="text-sm text-white/80">Manage your account and preferences</p>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-lg">
        <div className="border-b border-gray-200 bg-gradient-to-r from-gray-50 to-white px-6 py-4">
          <nav className="flex space-x-2">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-all ${
                  activeTab === tab.id
                    ? 'bg-gradient-to-r from-indigo-600 to-purple-600 text-white shadow-md'
                    : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
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
                <h3 className="text-lg font-bold text-gray-900">Personal Information</h3>
                <p className="text-sm text-gray-600">Update your personal details and athletic profile</p>
              </div>

              <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium text-gray-700">Full Name</label>
                  <input
                    type="text"
                    value={profileData.name}
                    onChange={(e) => setProfileData({ ...profileData, name: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-gray-300 px-4 py-2 text-sm transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700">Email Address</label>
                  <input
                    type="email"
                    value={profileData.email}
                    onChange={(e) => setProfileData({ ...profileData, email: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-gray-300 px-4 py-2 text-sm transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700">Weight (kg)</label>
                  <input
                    type="number"
                    value={profileData.weight}
                    onChange={(e) => setProfileData({ ...profileData, weight: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-gray-300 px-4 py-2 text-sm transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700">Max Heart Rate (bpm)</label>
                  <input
                    type="number"
                    value={profileData.maxHr}
                    onChange={(e) => setProfileData({ ...profileData, maxHr: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-gray-300 px-4 py-2 text-sm transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700">Resting Heart Rate (bpm)</label>
                  <input
                    type="number"
                    value={profileData.restingHr}
                    onChange={(e) => setProfileData({ ...profileData, restingHr: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-gray-300 px-4 py-2 text-sm transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
                  />
                </div>
              </div>

              <div className="flex items-center gap-3">
                <button
                  onClick={handleSaveProfile}
                  disabled={isSaving}
                  className="flex items-center gap-2 rounded-xl bg-gradient-to-r from-indigo-600 to-purple-600 px-6 py-3 text-sm font-bold text-white shadow-md transition hover:shadow-lg disabled:opacity-50"
                >
                  <Save className="h-4 w-4" />
                  {isSaving ? 'Saving...' : 'Save Changes'}
                </button>
                {saveSuccess && (
                  <span className="text-sm font-medium text-emerald-600">Changes saved successfully!</span>
                )}
              </div>
            </div>
          )}

          {/* Notifications Tab */}
          {activeTab === 'notifications' && (
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-bold text-gray-900">Notification Preferences</h3>
                <p className="text-sm text-gray-600">Choose what updates you want to receive</p>
              </div>

              <div className="space-y-4">
                {Object.entries({
                  workoutReminders: 'Daily workout reminders',
                  weeklyReport: 'Weekly training summary',
                  achievementAlerts: 'Achievement and milestone alerts',
                  coachMessages: 'Messages from your AI coach',
                }).map(([key, label]) => (
                  <div key={key} className="flex items-center justify-between rounded-xl bg-gray-50 p-4">
                    <span className="text-sm font-medium text-gray-900">{label}</span>
                    <button
                      onClick={() => setNotifications({ ...notifications, [key]: !notifications[key as keyof typeof notifications] })}
                      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                        notifications[key as keyof typeof notifications] ? 'bg-indigo-600' : 'bg-gray-300'
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
                <h3 className="text-lg font-bold text-gray-900">Connected Services</h3>
                <p className="text-sm text-gray-600">Manage your third-party integrations</p>
              </div>

              <div className="space-y-4">
                {['Strava', 'Garmin Connect', 'Apple Health', 'Google Fit'].map((service) => (
                  <div key={service} className="flex items-center justify-between rounded-xl border border-gray-200 bg-gradient-to-r from-gray-50 to-white p-5 shadow-sm">
                    <div>
                      <p className="font-semibold text-gray-900">{service}</p>
                      <p className="text-xs text-gray-500">
                        {service === 'Strava' ? 'Connected' : 'Not connected'}
                      </p>
                    </div>
                    <button
                      className={`rounded-lg px-4 py-2 text-sm font-semibold transition ${
                        service === 'Strava'
                          ? 'bg-rose-100 text-rose-700 hover:bg-rose-200'
                          : 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200'
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
                <h3 className="text-lg font-bold text-gray-900">Privacy & Security</h3>
                <p className="text-sm text-gray-600">Control your data and privacy settings</p>
              </div>

              <div className="space-y-4">
                <div className="rounded-xl border border-gray-200 bg-gradient-to-r from-blue-50 to-indigo-50 p-5">
                  <h4 className="font-semibold text-gray-900">Data Export</h4>
                  <p className="mt-1 text-sm text-gray-600">Download all your training data</p>
                  <button className="mt-3 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-indigo-700">
                    Request Export
                  </button>
                </div>

                <div className="rounded-xl border border-gray-200 bg-gradient-to-r from-amber-50 to-orange-50 p-5">
                  <h4 className="font-semibold text-gray-900">Password</h4>
                  <p className="mt-1 text-sm text-gray-600">Change your account password</p>
                  <button className="mt-3 rounded-lg bg-amber-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-amber-700">
                    Update Password
                  </button>
                </div>

                <div className="rounded-xl border border-rose-200 bg-gradient-to-r from-rose-50 to-red-50 p-5">
                  <h4 className="font-semibold text-rose-900">Delete Account</h4>
                  <p className="mt-1 text-sm text-rose-700">Permanently delete your account and all data</p>
                  <button className="mt-3 rounded-lg bg-rose-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-rose-700">
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
